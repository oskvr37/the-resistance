import JoinCreateRoom from "@/components/JoinCreateRoom";

import { useAuthStore } from "@/store/auth";
import { useEffect } from "react";

function App() {
  const checkAuth = useAuthStore((state) => state.checkAuth);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  return (
    <main>
      <section>
        <JoinCreateRoom />
      </section>
    </main>
  );
}

export default App;
