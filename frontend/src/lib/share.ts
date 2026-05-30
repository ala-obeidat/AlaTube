const DB_NAME = 'alatube-share';
const STORE = 'pending';
const KEY = 'latest';

export type PendingShare = {
  text: string;
  createdAt: number;
};

export async function readPendingShare(): Promise<PendingShare | null> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE, 'readwrite');
    const store = tx.objectStore(STORE);
    const req = store.get(KEY);
    req.onerror = () => reject(req.error);
    req.onsuccess = () => {
      const value = (req.result ?? null) as PendingShare | null;
      store.delete(KEY);
      resolve(value);
    };
  });
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => req.result.createObjectStore(STORE);
    req.onerror = () => reject(req.error);
    req.onsuccess = () => resolve(req.result);
  });
}

