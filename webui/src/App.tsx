import React from "react"
import { ToastContainer } from "react-toastify"
import 'react-toastify/dist/ReactToastify.css';
import { BrowserRouter as Router, Route, Routes } from "react-router-dom"
import { TableSettingsContextProvider } from "./context/TableSettingsContext"
import { Home } from "./pages/Home/Home"
import { JobsDashboard } from "./pages/JobsDashboard/JobsDashboard"
import { NodesDashboard } from "./pages/NodesDashboard/NodesDashboard"
import { Settings } from "./pages/Settings/Settings"
import { JobDetail } from "./pages/JobDetail/JobDetail"
import { Flow } from "./pages/Auth/Flow";

const App = () => (
  <TableSettingsContextProvider>
    <ToastContainer position="top-center"/>
    <Router>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/JobsDashboard" element={<JobsDashboard />} />
        <Route path="/JobDetail/:jobId" element={<JobDetail />} />
        <Route path="/NodesDashboard" element={<NodesDashboard />} />
        <Route path="/Auth" element={<Flow />} />
        <Route path="/Settings" element={<Settings />} />
      </Routes>
    </Router>
  </TableSettingsContextProvider>
)

export default App
