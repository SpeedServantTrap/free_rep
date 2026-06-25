import { useState, useCallback, useEffect } from 'react'
import { Shield, Copy, Check, Play, Square } from 'lucide-react'
import { useStore }         from '@/store'
import { useSend }          from '@/hooks/useWebSocket'
import { Button, Badge, Card, EmptyState, ScanAnimation } from '@/components/ui'

const SERVICE = 'nmap_service'

export default function NmapScanner() {
  const [input,  setInput]  = useState('')
  const [copied, setCopied] = useState(false)
  const [pendingCommand, setPendingCommand] = useState(null)

  const send         = useSend()
  const activeScan   = useStore((s) => s.activeScan)
  const latestResult = useStore((s) => s.latestResult)
  const wsStatus     = useStore((s) => s.wsStatus)
  const autoScanRunning = useStore((s) => s.nmapAutoScanRunning)
  const setAutoScanRunning = useStore((s) => s.setNmapAutoScanRunning)

  const isScanning  = activeScan?.scanner_service === SERVICE && activeScan?.options?.scan_method === 'comprehensive_scan' && !activeScan?.options?.command
  const isConnected = wsStatus === 'connected'
  const result     = latestResult?.scanner_service === SERVICE ? latestResult : null
  const scanResult = result?.result

  useEffect(() => {
    if (pendingCommand && result?.result?.status) {
      const status = result.result.status

      if (pendingCommand === 'start' && status === 'started') {
        setAutoScanRunning(true)
        setPendingCommand(null)
      } else if (pendingCommand === 'stop' && status === 'stopped') {
        setAutoScanRunning(false)
        setPendingCommand(null)
      } else if (status === 'failed') {
        setPendingCommand(null)
      }
    }
  }, [result, pendingCommand, setAutoScanRunning])

  const handleScan = useCallback(() => {
    if (!input || isScanning) return
    send(SERVICE, { scan_method: 'comprehensive_scan', input })
  }, [input, isScanning, send])

  const handleStartAutoScan = useCallback(() => {
    if (!isConnected || autoScanRunning || pendingCommand) return
    send(SERVICE, { command: 'start' })
    setPendingCommand('start')
  }, [isConnected, autoScanRunning, pendingCommand, send])

  const handleStopAutoScan = useCallback(() => {
    if (!isConnected || !autoScanRunning || pendingCommand) return
    send(SERVICE, { command: 'stop' })
    setPendingCommand('stop')
  }, [isConnected, autoScanRunning, pendingCommand, send])

  const handleKey = (e) => {
    if (e.ctrlKey && e.key === 'Enter') handleScan()
  }

  const copyJSON = () => {
    navigator.clipboard.writeText(JSON.stringify(scanResult, null, 2))
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div onKeyDown={handleKey}>
      <div className="page-header">
        <div>
          <h1 className="page-title"><Shield size={22} /> Nmap Scanner</h1>
          <p className="page-subtitle">Unified TCP 65k, UDP 65k, OS and DNS scan for one IP, comma-separated IPs or CIDR</p>
        </div>
      </div>

      <div style={{ display: 'grid', gap: 20 }}>
        <Card title="Scan Configuration">
          <div className="form-group">
            <label>Targets</label>
            <input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="192.168.1.10 or 192.168.1.10,192.168.1.20 or 192.168.1.0/24"
            />
          </div>

          <div style={{ marginTop: 16, display: 'flex', gap: 10, alignItems: 'center' }}>
            <Button variant="primary" size="lg" loading={isScanning} onClick={handleScan} disabled={!input || !isConnected}>
              {isScanning ? 'Scanning…' : 'Run Comprehensive Scan'}
            </Button>
            <span className="text-muted text-sm">{isConnected ? 'Ctrl + Enter' : 'Waiting for backend…'}</span>
          </div>
        </Card>

        <Card title="Auto Scan Control">
          <div style={{ marginTop: 16, display: 'flex', gap: 10, alignItems: 'center' }}>
            <Button
              variant="primary"
              size="lg"
              loading={pendingCommand === 'start'}
              onClick={handleStartAutoScan}
              disabled={!isConnected || autoScanRunning || pendingCommand !== null}
            >
              <Play size={16} style={{ marginRight: 8 }} />
              {pendingCommand === 'start' ? 'Starting…' : autoScanRunning ? 'Auto Scan Running…' : 'Start Auto Scan'}
            </Button>
            <Button
              variant="danger"
              size="lg"
              loading={pendingCommand === 'stop'}
              onClick={handleStopAutoScan}
              disabled={!isConnected || !autoScanRunning || pendingCommand !== null}
            >
              <Square size={16} style={{ marginRight: 8 }} />
              {pendingCommand === 'stop' ? 'Stopping…' : 'Stop Auto Scan'}
            </Button>
            <span className="text-muted text-sm">
              {isConnected ? 'Auto scan targets and interval come from scanner_nmap .env' : 'Waiting for backend…'}
            </span>
          </div>
        </Card>

        {isScanning && (
          <Card>
            <ScanAnimation label={`Running comprehensive scan on ${input}…`} />
          </Card>
        )}

        {(autoScanRunning || pendingCommand === 'start') && (
          <Card>
            <ScanAnimation label="Auto Nmap scan running… Results will be saved continuously." />
          </Card>
        )}

        {scanResult && !isScanning && (
          <div className="result-panel animate-in">
            <div className="result-header">
              <span className="result-title"><Shield size={14} /> Nmap Result</span>
              <div style={{ display: 'flex', gap: 8 }}>
                <Badge>{scanResult.status ?? 'completed'}</Badge>
                <button className="copy-btn" onClick={copyJSON}>
                  {copied ? <Check size={14} /> : <Copy size={14} />}
                </button>
              </div>
            </div>

            {scanResult.error ? (
              <div className="result-error">Error: {scanResult.error}</div>
            ) : (
              <ComprehensiveResult r={scanResult} />
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function ComprehensiveResult({ r }) {
  if (!r.results?.length) return <EmptyState title="No hosts found" />

  return (
    <div style={{ display: 'grid', gap: 16 }}>
      <div className="result-stats">
        <div className="result-stat">
          <div className="result-stat-value">{r.results.length}</div>
          <div className="result-stat-label">Hosts</div>
        </div>
        <div className="result-stat">
          <div className="result-stat-value">{countOpenPorts(r.results, 'tcp')}</div>
          <div className="result-stat-label">Open TCP</div>
        </div>
        <div className="result-stat">
          <div className="result-stat-value">{countOpenPorts(r.results, 'udp')}</div>
          <div className="result-stat-label">Open UDP</div>
        </div>
      </div>

      {r.results.map((target, index) => {
        const tcpPorts = flattenPorts(target.tcp_port_info)
        const udpPorts = flattenPorts(target.udp_port_info)

        return (
          <Card key={`${target.host}-${index}`} title={target.host || `Target ${index + 1}`}>
            <div className="os-result-grid" style={{ marginBottom: 16 }}>
              {[
                { k: 'DNS', v: target.dns || '—' },
                { k: 'Discovery', v: target.discovery_status || '—' },
                { k: 'Reason', v: target.discovery_reason || '—' },
                { k: 'OS Name', v: target.os_name || '—' },
                { k: 'Vendor', v: target.os_vendor || '—' },
                { k: 'Accuracy', v: target.os_accuracy ? `${target.os_accuracy}%` : '—' },
              ].map(({ k, v }) => (
                <div key={k} className="os-result-item">
                  <div className="os-result-key">{k}</div>
                  <div className="os-result-value">{v}</div>
                </div>
              ))}
            </div>

            {(target.tcp_error || target.udp_error || target.os_error || target.dns_error) && (
              <div className="result-error" style={{ marginBottom: 16 }}>
                {[target.tcp_error, target.udp_error, target.os_error, target.dns_error].filter(Boolean).join(' · ')}
              </div>
            )}

            <PortTable title="TCP Ports" ports={tcpPorts} emptyTitle="No TCP ports found" />
            <PortTable title="UDP Ports" ports={udpPorts} emptyTitle="No UDP ports found" />
          </Card>
        )
      })}
    </div>
  )
}

function PortTable({ title, ports, emptyTitle }) {
  return (
    <div style={{ marginTop: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <strong>{title}</strong>
        <Badge>{ports.length}</Badge>
      </div>
      {!ports.length ? (
        <EmptyState title={emptyTitle} />
      ) : (
        <div className="table-wrap">
          <table>
            <thead><tr>
              <th>Port</th>
              <th>Protocol</th>
              <th>State</th>
              <th>Service</th>
            </tr></thead>
            <tbody>
              {ports.map((port, index) => (
                <tr key={`${port.protocol}-${port.port}-${index}`}>
                  <td className="td-mono" style={{ color: isOpenState(port.state) ? 'var(--green)' : undefined }}>{port.port}</td>
                  <td className="td-mono td-muted">{port.protocol}</td>
                  <td><Badge>{port.state}</Badge></td>
                  <td className="td-muted">{port.serviceName}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function flattenPorts(portInfo) {
  const ports = []

  portInfo?.forEach((info) => {
    info.all_ports?.forEach((port, index) => {
      ports.push({
        port,
        protocol: info.protocols?.[index] ?? '—',
        state: info.state?.[index] ?? '—',
        serviceName: info.service_name?.[index] ?? '—',
      })
    })
  })

  return ports
}

function countOpenPorts(results, protocol) {
  return results.reduce((total, target) => {
    const list = protocol === 'tcp' ? flattenPorts(target.tcp_port_info) : flattenPorts(target.udp_port_info)
    return total + list.filter((port) => isOpenState(port.state)).length
  }, 0)
}

function isOpenState(state) {
  return state === 'open' || state === 'open|filtered'
}

