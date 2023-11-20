// src/_app.tsx

import React from "react";
import type { AppProps } from "next/app";
import { TableSettingsProvider } from "../context/TableSettingsContext";

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <TableSettingsProvider>
      <Component {...pageProps} />;
    </TableSettingsProvider>
  );
}

export default MyApp;
