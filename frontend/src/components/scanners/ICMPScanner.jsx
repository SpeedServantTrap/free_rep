import { useState, useCallback, useEffect } from 'react'
import { Radio, Copy, Check, Play, Square } from 'lucide-react'
import { useStore }         from '@/store'
import { useSend }          from '@/hooks/useWebSocket'
import { Button, Badge, Card, EmptyState, ScanAnimation } from '@/components/ui'

const SERVICE = 'icmp_service'

export default function ICMPScanner() {
  const [copied, setCopied] = useState(false)
  const [pendingCommand, setPendingCommand] = useState(null) // 'start' or 'stop'

  const send                = useSend()
  const latestResult        = useStore((s) => s.latestResult)
  const wsStatus            = useStore((s) => s.wsStatus)
  const autoScanRunning     = useStore((s) => s.icmpAutoScanRunning)
  const setAutoScanRunning  = useStore((s) => s.setIcmpAutoScanRunning)

  const isConnected = wsStatus === 'connected'
  const result     = latestResult?.scanner_service === SERVICE ? latestResult : null
  const scanResult = result?.result

  // Handle control command responses
  useEffect(() => {
    if (pendingCommand && result?.result?.status) {
      const status = result.result.status
      console.log('[ICMPScanner] Received response for pending command:', pendingCommand, 'status:', status)

      if (pendingCommand === 'start' && status === 'started') {
        setAutoScanRunning(true)
        setPendingCommand(null)
      } else if (pendingCommand === 'stop' && status === 'stopped') {
        setAutoScanRunning(false)
        setPendingCommand(null)
      } else if (status === 'failed') {
        console.error('[ICMPScanner] Command failed:', result.result.error)
        setPendingCommand(null)
      }
    }
  }, [result, pendingCommand, setAutoScanRunning])

  const handleStartAutoScan = useCallback(() => {
    console.log('[ICMPScanner] Start button clicked, autoScanRunning:', autoScanRunning, 'isConnected:', isConnected)
    if (!isConnected || autoScanRunning || pendingCommand) return
    console.log('[ICMPScanner] Sending start command')
    send(SERVICE, { command: 'start' })
    setPendingCommand('start')
  }, [isConnected, autoScanRunning, pendingCommand, send])

  const handleStopAutoScan = useCallback(() => {
    console.log('[ICMPScanner] Stop button clicked, autoScanRunning:', autoScanRunning, 'isConnected:', isConnected)
    if (!isConnected || !autoScanRunning || pendingCommand) return
    console.log('[ICMPScanner] Sending stop command')
    send(SERVICE, { command: 'stop' })
    setPendingCommand('stop')
  }, [isConnected, autoScanRunning, pendingCommand, send])

  const copyJSON = () => {
    navigator.clipboard.writeText(JSON.stringify(scanResult, null, 2))
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const lossColor = (pct) => {
    if (pct === 0)   return 'var(--green)'
    if (pct < 50)    return 'var(--yellow)'
    return 'var(--red)'
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title"><Radio size={22} /> ICMP Scanner</h1>
          <p className="page-subtitle">Auto-scan configured in scanner .env file</p>
        </div>
      </div>

      <div style={{ display: 'grid', gap: 20 }}>
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
              {isConnected ? 'Auto scan configured in scanner .env file' : 'Waiting for backend…'}
            </span>
          </div>
        </Card>

        {(autoScanRunning || pendingCommand === 'start') && (
          <Card>
            <ScanAnimation label="Auto scan running… Results will appear when available." />
          </Card>
        )}

        {scanResult && (
          <div className="result-panel animate-in">
            <div className="result-header">
              <span className="result-title"><Radio size={14} /> ICMP Scan Result</span>
              <div style={{ display: 'flex', gap: 8 }}>
                <Badge>{scanResult.status ?? 'completed'}</Badge>
                <button className="copy-btn" onClick={copyJSON}>
                  {copied ? <Check size={14} /> : <Copy size={14} />}
                </button>
              </div>
            </div>

            {scanResult.error ? (
              <div className="result-error">Error: {scanResult.error}</div>
            ) : scanResult.results?.length > 0 ? (
              <div className="table-wrap">
                <table>
                  <thead><tr>
                    <th>Target</th>
                    <th>Address</th>
                    <th style={{ textAlign: 'right' }}>Sent</th>
                    <th style={{ textAlign: 'right' }}>Received</th>
                    <th style={{ textAlign: 'right' }}>Loss</th>
                    <th>Status</th>
                  </tr></thead>
                  <tbody>
                    {scanResult.results.map((r, i) => (
                      <tr key={i}>
                        <td className="td-mono">{r.target}</td>
                        <td className="td-mono td-muted">{r.address || '—'}</td>
                        <td style={{ textAlign: 'right', fontFamily: 'var(--font-mono)' }}>{r.packets_sent}</td>
                        <td style={{ textAlign: 'right', fontFamily: 'var(--font-mono)' }}>{r.packets_received}</td>
                        <td style={{ textAlign: 'right', fontFamily: 'var(--font-mono)', color: lossColor(r.packet_loss_percent) }}>
                          {r.packet_loss_percent?.toFixed(1)}%
                        </td>
                        <td>
                          <Badge>{r.error ? 'error' : r.packet_loss_percent === 0 ? 'online' : r.packet_loss_percent >= 100 ? 'offline' : 'partial'}</Badge>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <EmptyState title="No results" />
            )}
          </div>
        )}
      </div>
    </div>
  )
}

