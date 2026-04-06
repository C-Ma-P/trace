import * as THREE from 'three';

// occt-import-js is a CommonJS module with WASM; we dynamically import it.
// The WASM file is copied to the build output by vite-plugin-static-copy.

interface OcctMeshAttribute {
  array: number[];
}

interface OcctMesh {
  name: string;
  color?: [number, number, number];
  attributes: {
    position: OcctMeshAttribute;
    normal?: OcctMeshAttribute;
  };
  index: {
    array: number[];
  };
}

interface OcctResult {
  success: boolean;
  meshes: OcctMesh[];
  root: {
    name: string;
    meshes: number[];
    children: any[];
  };
}

interface OcctInstance {
  ReadStepFile(content: Uint8Array, params: any): OcctResult;
}

let occtPromise: Promise<OcctInstance> | null = null;

function getOcct(): Promise<OcctInstance> {
  if (!occtPromise) {
    occtPromise = (async () => {
      // Dynamic import for the occt-import-js module
      const occtModule = await import('occt-import-js');
      const initFn = occtModule.default || occtModule;
      const occt: OcctInstance = await initFn({
        locateFile: (name: string) => {
          if (name.endsWith('.wasm')) {
            // In dev mode, Vite serves from the copied location
            return `/occt-import-js.wasm`;
          }
          return name;
        },
      });
      return occt;
    })();
  }
  return occtPromise;
}

/**
 * Parse STEP file bytes into Three.js BufferGeometry objects.
 * Returns a Group containing all meshes.
 */
export async function loadStepFileToGroup(fileBytes: Uint8Array): Promise<THREE.Group> {
  const occt = await getOcct();
  const result = occt.ReadStepFile(fileBytes, null);

  if (!result.success) {
    throw new Error('Failed to parse STEP file');
  }

  if (!result.meshes || result.meshes.length === 0) {
    throw new Error('STEP file contains no geometry');
  }

  const group = new THREE.Group();
  const defaultMaterial = new THREE.MeshStandardMaterial({
    color: 0x8899aa,
    metalness: 0.3,
    roughness: 0.6,
    side: THREE.DoubleSide,
  });

  for (const mesh of result.meshes) {
    const geometry = new THREE.BufferGeometry();

    const positions = new Float32Array(mesh.attributes.position.array);
    geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));

    if (mesh.attributes.normal) {
      const normals = new Float32Array(mesh.attributes.normal.array);
      geometry.setAttribute('normal', new THREE.BufferAttribute(normals, 3));
    } else {
      geometry.computeVertexNormals();
    }

    if (mesh.index && mesh.index.array.length > 0) {
      geometry.setIndex(new THREE.BufferAttribute(new Uint32Array(mesh.index.array), 1));
    }

    let material = defaultMaterial;
    if (mesh.color) {
      material = new THREE.MeshStandardMaterial({
        color: new THREE.Color(mesh.color[0], mesh.color[1], mesh.color[2]),
        metalness: 0.3,
        roughness: 0.6,
        side: THREE.DoubleSide,
      });
    }

    const threeMesh = new THREE.Mesh(geometry, material);
    group.add(threeMesh);
  }

  return group;
}

/**
 * Compute the bounding box of a group and return its center and size.
 */
export function getGroupBounds(group: THREE.Group): { center: THREE.Vector3; size: THREE.Vector3 } {
  const box = new THREE.Box3().setFromObject(group);
  const center = box.getCenter(new THREE.Vector3());
  const size = box.getSize(new THREE.Vector3());
  return { center, size };
}

/**
 * Dispose all geometries and materials in a group.
 */
export function disposeGroup(group: THREE.Group): void {
  group.traverse((child) => {
    if (child instanceof THREE.Mesh) {
      child.geometry.dispose();
      if (Array.isArray(child.material)) {
        child.material.forEach((m) => m.dispose());
      } else {
        child.material.dispose();
      }
    }
  });
}

/**
 * Decode a base64 string to a Uint8Array.
 */
export function base64ToUint8Array(base64: string): Uint8Array {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  return bytes;
}
