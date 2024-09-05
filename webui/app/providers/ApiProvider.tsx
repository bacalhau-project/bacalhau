'use client';

import { ReactNode, createContext, useContext, useState, useEffect } from 'react';
import { useApiInitialization } from '@/lib/api';

const ApiContext = createContext({ isInitialized: false });

export function ApiProvider({ children }: { children: ReactNode }) {
  const isInitialized = useApiInitialization();

  return (
    <ApiContext.Provider value={{ isInitialized }}>
      {children}
    </ApiContext.Provider>
  );
}

export function useApi() {
  return useContext(ApiContext);
}