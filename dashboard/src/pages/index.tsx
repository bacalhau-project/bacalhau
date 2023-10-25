// src/pages/index.tsx

import { useEffect } from "react";
import { useRouter } from "next/router";

const Home = () => {
  const router = useRouter();

  useEffect(() => {
    router.replace("/JobsDashboard");
  }, []);

  return null;
};

export default Home;
