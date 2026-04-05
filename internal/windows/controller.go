package windows

import (
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"componentmanager/internal/app"
)

// Controller manages window lifecycle for the application.
//
// On Linux GTK3, each window has its own physical menu bar embedded at creation
// time. Menus cannot be swapped or updated after window creation — SetMenu()
// only updates internal pointers without replacing the displayed GTK widget,
// and app.Menu.Set() is a no-op. Therefore, each window receives a freshly
// built menu at creation time and we never attempt to modify it afterward.
type Controller struct {
	app                  *application.App
	backend              *app.App
	mu                   sync.Mutex
	launcher             *application.WebviewWindow
	preferences          *application.WebviewWindow
	preferencesProjectID string
	projectIDs           map[string]struct{}
}

var traceWindowBackground = application.RGBA{Red: 0x0f, Green: 0x11, Blue: 0x17, Alpha: 0xff}

const (
	launcherViewDefault     = "launcher"
	launcherViewKiCadImport = "kicad-import"

	launcherDefaultWidth     = 1280
	launcherDefaultHeight    = 800
	launcherDefaultMinWidth  = 900
	launcherDefaultMinHeight = 600

	launcherKiCadWidth     = 1600
	launcherKiCadHeight    = 980
	launcherKiCadMinWidth  = 1280
	launcherKiCadMinHeight = 760
)

func NewController(appInstance *application.App, backend *app.App) *Controller {
	return &Controller{
		app:        appInstance,
		backend:    backend,
		projectIDs: map[string]struct{}{},
	}
}

func (c *Controller) ListOpenProjectIDs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.projectIDs))
	for id := range c.projectIDs {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func (c *Controller) showWindowOnRuntimeReady(window *application.WebviewWindow, focus bool, afterShow func()) {
	window.RegisterHook(events.Common.WindowRuntimeReady, func(e *application.WindowEvent) {
		window.Show()
		if focus {
			window.Focus()
		}
		if afterShow != nil {
			afterShow()
		}
	})
}

func (c *Controller) EnsureLauncherWindow() *application.WebviewWindow {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.launcher != nil {
		return c.launcher
	}

	launcherWindow := c.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "launcher",
		Title:            "Trace",
		Width:            1280,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: traceWindowBackground,
		Hidden:           true,
		URL:              c.withStartupStatus("/?mode=launcher"),
		Linux:            application.LinuxWindow{Menu: c.buildLauncherMenu()},
	})
	c.showWindowOnRuntimeReady(launcherWindow, true, nil)

	launcherWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		c.mu.Lock()
		hasProjects := len(c.projectIDs) > 0
		c.mu.Unlock()

		if hasProjects {
			e.Cancel()
			launcherWindow.Hide()
			return
		}
	})

	c.launcher = launcherWindow
	return launcherWindow
}

func (c *Controller) OpenPreferencesWindow(projectID string) *application.WebviewWindow {
	log.Printf("[PREFS] OpenPreferencesWindow called, projectID=%q", projectID)
	projectID = strings.TrimSpace(projectID)
	preferencesURL := c.preferencesURL(projectID)

	c.mu.Lock()

	// Recover stale reference: our pointer is nil but Wails still has the window.
	if c.preferences == nil {
		if existing, ok := c.app.Window.GetByName("preferences"); ok {
			if webview, ok := existing.(*application.WebviewWindow); ok {
				c.preferences = webview
			}
		}
	}

	// Detect destroyed window: our pointer is non-nil but Wails no longer has it.
	if c.preferences != nil {
		if _, ok := c.app.Window.GetByName("preferences"); !ok {
			c.preferences = nil
			c.preferencesProjectID = ""
		}
	}

	// Reuse existing window if same project context.
	if c.preferences != nil && c.preferencesProjectID == projectID {
		prefs := c.preferences
		c.mu.Unlock()
		prefs.SetURL(c.withStartupStatus(preferencesURL))
		prefs.Show()
		prefs.Focus()
		return prefs
	}

	// Close existing window if different project context.
	// We nil the reference first so the closing hook is a no-op.
	var oldPrefs *application.WebviewWindow
	if c.preferences != nil {
		oldPrefs = c.preferences
		c.preferences = nil
		c.preferencesProjectID = ""
	}
	c.mu.Unlock()

	if oldPrefs != nil {
		oldPrefs.Close()
	}

	// Create new preferences window with a fresh menu.
	preferencesWindow := c.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "preferences",
		Title:            "Trace Preferences",
		Width:            1040,
		Height:           760,
		MinWidth:         920,
		MinHeight:        600,
		BackgroundColour: traceWindowBackground,
		Hidden:           true,
		URL:              c.withStartupStatus(preferencesURL),
		Linux:            application.LinuxWindow{Menu: c.buildPreferencesMenu(projectID)},
	})
	c.showWindowOnRuntimeReady(preferencesWindow, true, nil)

	preferencesWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.preferences == preferencesWindow {
			c.preferences = nil
			c.preferencesProjectID = ""
		}
	})

	c.mu.Lock()
	// Guard against a concurrent call that may have created the window first.
	if c.preferences != nil {
		c.mu.Unlock()
		preferencesWindow.Close()
		return c.preferences
	}
	c.preferences = preferencesWindow
	c.preferencesProjectID = projectID
	c.mu.Unlock()

	return preferencesWindow
}

func (c *Controller) PickDirectory(startDir string) (string, error) {
	dialog := c.app.Dialog.OpenFile().
		SetTitle("Choose Folder").
		SetButtonText("Use This Folder").
		CanChooseDirectories(true).
		CanChooseFiles(false)

	trimmedStartDir := strings.TrimSpace(startDir)
	if trimmedStartDir != "" {
		dialog.SetDirectory(filepath.Clean(trimmedStartDir))
	}
	if current := c.app.Window.Current(); current != nil {
		dialog.AttachToWindow(current)
	}

	selectedDir, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(selectedDir), nil
}

func (c *Controller) PickFile(title string, filters ...application.FileFilter) (string, error) {
	dialog := c.app.Dialog.OpenFile().
		SetTitle(title).
		SetButtonText("Import").
		CanChooseDirectories(true).
		CanChooseFiles(true)

	for _, f := range filters {
		dialog.AddFilter(f.DisplayName, f.Pattern)
	}
	if current := c.app.Window.Current(); current != nil {
		dialog.AttachToWindow(current)
	}

	selected, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(selected), nil
}

func (c *Controller) SetLauncherView(view string) error {
	current := c.app.Window.Current()
	if current == nil {
		return fmt.Errorf("no active window")
	}

	launcherWindow, ok := current.(*application.WebviewWindow)
	if !ok {
		return fmt.Errorf("active window is not resizable")
	}
	if launcherWindow.Name() != "launcher" {
		return nil
	}

	width := launcherDefaultWidth
	height := launcherDefaultHeight
	minWidth := launcherDefaultMinWidth
	minHeight := launcherDefaultMinHeight

	switch strings.TrimSpace(view) {
	case "", launcherViewDefault:
	case launcherViewKiCadImport:
		width = launcherKiCadWidth
		height = launcherKiCadHeight
		minWidth = launcherKiCadMinWidth
		minHeight = launcherKiCadMinHeight
	default:
		return fmt.Errorf("unknown launcher view %q", view)
	}

	launcherWindow.SetSize(width, height)
	launcherWindow.SetMinSize(minWidth, minHeight)
	return nil
}

func (c *Controller) OpenProjectWindow(projectID string, hideLauncher bool) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return fmt.Errorf("project id required")
	}

	name := "project-" + projectID

	// Reuse existing project window — menu is already baked in from creation.
	if existing, ok := c.app.Window.GetByName(name); ok {
		existing.Show()
		existing.Focus()
		c.mu.Lock()
		c.projectIDs[projectID] = struct{}{}
		launcher := c.launcher
		c.mu.Unlock()
		if hideLauncher && launcher != nil {
			launcher.Hide()
		}
		return nil
	}

	projectURL := c.withStartupStatus("/?mode=project&projectId=" + url.QueryEscape(projectID))

	title := "Trace"
	for _, rp := range c.backend.ListRecentProjects() {
		if rp.ID == projectID && rp.Name != "" {
			title = "Trace — " + rp.Name
			break
		}
	}

	projectWindow := c.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             name,
		Title:            title,
		Width:            1280,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: traceWindowBackground,
		Hidden:           true,
		URL:              projectURL,
		Linux:            application.LinuxWindow{Menu: c.buildProjectMenu(projectID)},
	})
	c.showWindowOnRuntimeReady(projectWindow, true, func() {
		if hideLauncher {
			c.mu.Lock()
			launcher := c.launcher
			c.mu.Unlock()
			if launcher != nil {
				launcher.Hide()
			}
		}
	})

	c.mu.Lock()
	c.projectIDs[projectID] = struct{}{}
	c.mu.Unlock()

	projectWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		c.mu.Lock()
		delete(c.projectIDs, projectID)
		hasProjects := len(c.projectIDs) > 0
		launcher := c.launcher
		c.mu.Unlock()

		if !hasProjects {
			if launcher == nil {
				launcher = c.EnsureLauncherWindow()
			}
			launcher.Show()
			launcher.Focus()
		}
	})
	return nil
}

func (c *Controller) PromptOpenProjectWindow(projectID string) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return
	}

	current := c.app.Window.Current()
	if current == nil {
		if err := c.OpenProjectWindow(projectID, true); err != nil {
			log.Printf("Open Recent failed: %v", err)
		}
		return
	}

	currentName := current.Name()
	targetName := "project-" + projectID

	message := "Open this project in this window or a new window?"
	if currentName == "launcher" {
		message = "Open this project in the launcher window or a new window?"
	}

	dlg := c.app.Dialog.Question().
		SetTitle("Open Project").
		SetMessage(message).
		AttachToWindow(current)

	openInNewWindow := func(hideLauncher bool) {
		if err := c.OpenProjectWindow(projectID, hideLauncher); err != nil {
			c.app.Dialog.Error().
				SetTitle("Open Project Failed").
				SetMessage(err.Error()).
				AttachToWindow(current).
				Show()
		}
	}

	openInThisWindow := func() {
		if currentName == targetName {
			return
		}
		if err := c.OpenProjectWindow(projectID, true); err != nil {
			c.app.Dialog.Error().
				SetTitle("Open Project Failed").
				SetMessage(err.Error()).
				AttachToWindow(current).
				Show()
			return
		}
		if currentName != "launcher" {
			current.Close()
		} else {
			current.Hide()
		}
	}

	dlg.AddButton("This Window").OnClick(openInThisWindow).SetAsDefault()

	if currentName == "launcher" {
		dlg.AddButton("New Window").OnClick(func() {
			openInNewWindow(false)
		})
	} else {
		dlg.AddButton("New Window").OnClick(func() {
			openInNewWindow(true)
		})
	}

	dlg.AddButton("Cancel").SetAsCancel()
	dlg.Show()
}

// --- Menu builders ---
// Each call returns a fresh menu. On Linux GTK3, the menu is physically
// embedded into the window at creation time and cannot be changed afterward.

func (c *Controller) buildProjectMenu(projectID string) *application.Menu {
	menu := c.buildBaseFileMenu(true)
	editMenu := ensureSubmenu(menu, "Edit")
	editMenu.Add("Preferences").SetAccelerator("CmdOrCtrl+,").OnClick(func(ctx *application.Context) {
		log.Printf("[MENU] Edit > Preferences clicked (projectID=%s)", projectID)
		c.OpenPreferencesWindow(projectID)
	})
	return menu
}

func (c *Controller) buildLauncherMenu() *application.Menu {
	menu := c.buildBaseFileMenu(true)
	editMenu := ensureSubmenu(menu, "Edit")
	editMenu.Add("Preferences").SetAccelerator("CmdOrCtrl+,").OnClick(func(ctx *application.Context) {
		log.Printf("[MENU] Edit > Preferences clicked (launcher)")
		c.OpenPreferencesWindow("")
	})
	return menu
}

func (c *Controller) buildPreferencesMenu(projectID string) *application.Menu {
	menu := c.buildBaseFileMenu(false)
	if projectID != "" {
		editMenu := ensureSubmenu(menu, "Edit")
		editMenu.Add("Show Project Preferences").SetEnabled(false)
	}
	return menu
}

func (c *Controller) buildBaseFileMenu(includeOpenRecent bool) *application.Menu {
	menu := application.DefaultApplicationMenu()
	fileMenu := ensureSubmenu(menu, "File")

	if includeOpenRecent {
		openRecent := fileMenu.AddSubmenu("Open Recent")
		recents := c.backend.ListRecentProjects()
		if len(recents) == 0 {
			openRecent.Add("No recent projects").SetEnabled(false)
		} else {
			for _, rp := range recents {
				pid := rp.ID
				label := rp.Name
				if label == "" {
					label = pid
				}
				projectID := pid
				openRecent.Add(label).OnClick(func(ctx *application.Context) {
					c.PromptOpenProjectWindow(projectID)
				})
			}
		}
		fileMenu.AddSeparator()
	}

	fileMenu.Add("Close Window").SetAccelerator("CmdOrCtrl+W").OnClick(func(ctx *application.Context) {
		if current := c.app.Window.Current(); current != nil {
			current.Close()
		}
	})

	return menu
}

func ensureSubmenu(menu *application.Menu, label string) *application.Menu {
	item := menu.FindByLabel(label)
	if item != nil {
		if submenu := item.GetSubmenu(); submenu != nil {
			return submenu
		}
	}
	return menu.AddSubmenu(label)
}

func (c *Controller) withStartupStatus(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	query := parsedURL.Query()
	query.Set("startup", c.startupStatusValue())
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

func (c *Controller) startupStatusValue() string {
	if c.backend == nil {
		return "unknown"
	}
	if c.backend.GetStartupStatus().Ready {
		return "ready"
	}
	return "failed"
}

func (c *Controller) preferencesURL(projectID string) string {
	if projectID == "" {
		return "/?mode=preferences"
	}
	return "/?mode=preferences&projectId=" + url.QueryEscape(projectID)
}
