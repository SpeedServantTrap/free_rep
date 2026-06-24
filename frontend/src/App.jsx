import { Routes, Route, Navigate } from 'react-router-dom'
import Layout       from './components/layout/Layout'
import Dashboard    from './components/pages/Dashboard'
import ARPScanner   from './components/scanners/ARPScanner'
import ICMPScanner  from './components/scanners/ICMPScanner'
import NmapScanner  from './components/scanners/NmapScanner'
import HistoryPage  from './components/pages/HistoryPage'
import SearchPage   from './components/pages/SearchPage'
import ChangesPage  from './components/pages/ChangesPage'

export default function App() {
  return (
    <Routes>
      <Route path="/search" element={<SearchPage />} />
      <Route element={<Layout />}>
        <Route path="/" element={<Dashboard />} />
        <Route path="/arp" element={<ARPScanner />} />
        <Route path="/icmp" element={<ICMPScanner />} />
        <Route path="/nmap" element={<NmapScanner />} />
        <Route path="/history" element={<HistoryPage />} />
        <Route path="/changes" element={<ChangesPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}
