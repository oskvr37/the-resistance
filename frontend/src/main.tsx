import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router";
import "@/index.css";

import Layout from "@/components/Layout";
import LoginForm from "@/components/LoginForm";
import JoinCreateRoom from "@/components/JoinCreateRoom";
import Room from "./components/Room";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<JoinCreateRoom />} />
          <Route path="/login" element={<LoginForm />} />
          <Route path="/room/:room_id" element={<Room />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>,
);
