'use client';

import { useState, useEffect } from 'react';

const DEVICE_ID_KEY = 'miru_device_id';

export function useDeviceId(): string {
  const [deviceId, setDeviceId] = useState<string>('');

  useEffect(() => {
    if (typeof window === 'undefined') return;

    let id = localStorage.getItem(DEVICE_ID_KEY);
    
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem(DEVICE_ID_KEY, id);
    }
    
    setDeviceId(id);
  }, []);

  return deviceId;
}
