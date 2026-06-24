import { Outlet } from 'react-router-dom'
import { useWebSocket } from '@/hooks/useWebSocket'
import Sidebar from './Sidebar'

export default function Layout() {
  useWebSocket()

  return (
    <div className="layout">
      <Sidebar />
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  )
}
