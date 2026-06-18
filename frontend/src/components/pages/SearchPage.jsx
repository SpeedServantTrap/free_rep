import { useState } from 'react'
import { Search, Network, Shield, Globe, Server, Clock, Activity } from 'lucide-react'
import { api }           from '@/api/http'
import { Button, Badge, EmptyState, Card, Spinner } from '@/components/ui'
import toast             from 'react-hot-toast'

export default function SearchPage() {
  const [results, setResults] = useState(null)
  const [loading, setLoading] = useState(false)

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title"><Search size={22} /> Device Search</h1>
          <p className="page-subtitle">Search devices by MAC address or IP address</p>
        </div>
      </div>

      <NewDeviceSearch onResult={setResults} loading={loading} setLoading={setLoading} />
    </div>
  )
}

function NewDeviceSearch({ onResult, loading, setLoading }) {
  const [query, setQuery] = useState('')
  const [searchResult, setSearchResult] = useState(null)

  const handleUniversalSearch = async () => {
    if (!query.trim()) return

    setLoading(true)
    setSearchResult(null)
    try {
      const res = await api.universalSearch(query)
      if (!res.success) {
        toast.error(res.error ?? 'Search failed')
        onResult(res)
      } else {
        setSearchResult(res)
        onResult(res)
      }
    } catch (err) {
      console.error('Search error:', err)
      toast.error('Search failed')
    } finally {
      setLoading(false)
    }
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter') {
      handleUniversalSearch()
    }
  }

  return (
    <div>
      <Card style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 16 }}>
          <div style={{
            flex: 1,
            display: 'flex',
            gap: 0,
            alignItems: 'stretch',
            borderRadius: 8,
            overflow: 'hidden',
            border: '1px solid var(--border)',
          }}>
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="mac: xx:xx:xx:xx:xx:xx or ip: x.x.x.x"
              style={{
                flex: 1,
                padding: '12px 16px',
                border: 'none',
                outline: 'none',
                fontSize: 14,
                fontFamily: 'monospace',
              }}
            />
            <Button
              variant="primary"
              loading={loading}
              onClick={handleUniversalSearch}
              disabled={!query.trim()}
              style={{
                borderRadius: 0,
                borderTopLeftRadius: 0,
                borderBottomLeftRadius: 0,
                padding: '12px 24px',
              }}
            >
              <Search size={16} />
            </Button>
          </div>
        </div>
        <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
          Format: <code style={{ background: 'var(--bg-secondary)', padding: '2px 6px', borderRadius: 4 }}>mac: xx:xx:xx:xx:xx:xx</code> or <code style={{ background: 'var(--bg-secondary)', padding: '2px 6px', borderRadius: 4 }}>ip: x.x.x.x</code>
        </div>
      </Card>

      {loading && <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}><Spinner size="lg" /></div>}
      {!loading && searchResult === null && <EmptyState title="Device Search" description="Enter a MAC or IP address to search" />}
      {!loading && searchResult !== null && !searchResult.found && <EmptyState title="No results found" description="Try a different MAC or IP address" />}
      {!loading && searchResult?.found && (
        <DeviceSearchResult data={searchResult.data} />
      )}
    </div>
  )
}

function DeviceSearchResult({ data }) {
  if (!data) return null

  // Check if it's L2 or L3 device based on structure
  // API returns 'id' field (not '_id'), so we check both
  // L3 devices have IP address in id field (contains dots) or ip field
  // L2 devices have MAC address in id field (contains colons) without IP
  const idField = data.id || data._id
  const isL3 = idField && (idField.includes('.') || data.ip !== undefined)
  const isL2 = !isL3 && idField && idField.includes(':')

  if (isL3) {
    return <L3DeviceCard device={data} />
  } else if (isL2) {
    return <L2DeviceCard device={data} />
  }

  return <EmptyState title="Unknown device type" />
}

function L2DeviceCard({ device }) {
  const hasIpAddresses = device.ip_addresses?.length > 0
  const hasScanTimes = device.scan_times?.length > 0
  const hasScanners = device.scanner_types?.length > 0

  // Use 'id' field (MAC address) or fallback to 'mac' or '_id'
  const displayId = device.id || device.mac || device._id

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Header Section */}
      <Card style={{
        background: 'linear-gradient(135deg, #134e4a 0%, #0f172a 100%)',
        border: '1px solid rgba(34, 197, 94, 0.3)',
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 20 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <div style={{
              width: 56,
              height: 56,
              borderRadius: 14,
              background: 'rgba(34, 197, 94, 0.3)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}>
              <Network size={28} style={{ color: '#86efac' }} />
            </div>
            <div>
              <div style={{ fontSize: 24, fontWeight: 700, color: '#fff', marginBottom: 4, fontFamily: 'monospace' }}>
                {displayId}
              </div>
              <div style={{ fontSize: 16, color: '#86efac', fontWeight: 700 }}>
                L2 Device
              </div>
            </div>
          </div>
          <Badge dot={false} style={{ background: 'rgba(34, 197, 94, 0.3)', color: '#86efac', fontWeight: 700, fontSize: 14, padding: '6px 12px' }}>
            Active
          </Badge>
        </div>

        {/* Main Information Grid */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 12, marginBottom: 20 }}>
          <DeviceInfoRow icon={Globe} label="Vendor" value={device.vendor || 'Unknown'} />
          <DeviceInfoRow icon={Clock} label="First Seen" value={device.first_seen ? new Date(device.first_seen).toLocaleString() : 'N/A'} />
          <DeviceInfoRow icon={Activity} label="Last Seen" value={device.last_seen ? new Date(device.last_seen).toLocaleString() : 'N/A'} />
        </div>
      </Card>

      {/* IP Addresses Section */}
      {hasIpAddresses && (
        <Card style={{ background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)', border: '1px solid rgba(59, 130, 246, 0.3)' }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#93c5fd', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Server size={20} />
            IP Addresses
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.ip_addresses?.map((ip, i) => (
              <Badge key={i} style={{
                background: 'rgba(59, 130, 246, 0.3)',
                color: '#bfdbfe',
                border: '1px solid rgba(59, 130, 246, 0.5)',
                fontFamily: 'monospace',
                fontWeight: 700,
                fontSize: 13
              }}>
                {ip}
              </Badge>
            ))}
          </div>
        </Card>
      )}

      {/* Scan Times Section */}
      {hasScanTimes && (
        <Card style={{ background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)', border: '1px solid rgba(99, 102, 241, 0.3)' }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#c7d2fe', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Clock size={20} />
            Scan Times
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.scan_times.slice(0, 10).map((time, i) => (
              <span key={i} style={{
                fontSize: 12,
                padding: '6px 10px',
                borderRadius: 4,
                background: 'rgba(99, 102, 241, 0.3)',
                color: '#e0e7ff',
                fontFamily: 'monospace',
                fontWeight: 600
              }}>
                {new Date(time).toLocaleString()}
              </span>
            ))}
            {device.scan_times.length > 10 && (
              <span style={{ fontSize: 12, color: '#c7d2fe', fontWeight: 700 }}>
                +{device.scan_times.length - 10} more...
              </span>
            )}
          </div>
        </Card>
      )}

      {/* Scanner Types Section */}
      {hasScanners && (
        <Card style={{ background: 'linear-gradient(135deg, #1e293b 0%, #0f172a 100%)', border: '1px solid rgba(239, 68, 68, 0.3)' }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#fca5a5', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Shield size={20} />
            Scanner Types
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.scanner_types?.map((scanner, i) => (
              <Badge key={i} style={{
                background: 'rgba(239, 68, 68, 0.3)',
                color: '#fecaca',
                border: '1px solid rgba(239, 68, 68, 0.5)',
                fontWeight: 700,
                textTransform: 'uppercase',
                fontSize: 12
              }}>
                {scanner}
              </Badge>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}

function L3DeviceCard({ device }) {
  const hasPorts = (device.tcp_open_ports?.length > 0) || (device.udp_open_ports?.length > 0)
  const hasPackets = device.packets_reached?.length > 0
  const hasBanner = device.tcp_banner && device.tcp_banner !== '-'
  const hasScanTimes = device.scan_times?.length > 0
  const hasScanners = device.scanner_types?.length > 0

  // Use 'id' field (IP address) or fallback to 'ip' or '_id'
  const displayId = device.id || device.ip || device._id

  return (
    <Card style={{
      background: 'linear-gradient(135deg, #1e3a5f 0%, #0f172a 100%)',
      border: '1px solid rgba(59, 130, 246, 0.3)',
    }}>
      {/* Header Section */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <div style={{
            width: 64,
            height: 64,
            borderRadius: 16,
            background: 'rgba(59, 130, 246, 0.3)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <Server size={32} style={{ color: '#93c5fd' }} />
          </div>
          <div>
            <div style={{ fontSize: 24, fontWeight: 700, color: '#fff', marginBottom: 6, fontFamily: 'monospace' }}>
              {displayId}
            </div>
            <div style={{ fontSize: 16, color: '#93c5fd', fontWeight: 700 }}>
              L3 Device
            </div>
          </div>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, alignItems: 'flex-end' }}>
          <Badge dot={false} style={{ background: 'rgba(34, 197, 94, 0.3)', color: '#86efac', fontWeight: 700, fontSize: 14, padding: '6px 12px' }}>
            Active
          </Badge>
          {device.mac && (
            <Badge dot={false} style={{ background: 'rgba(168, 85, 247, 0.4)', color: '#ddd6fe', fontFamily: 'monospace', fontWeight: 700, fontSize: 15, padding: '8px 12px' }}>
              {device.mac}
            </Badge>
          )}
        </div>
      </div>

      {/* Main Information Grid */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 16, marginBottom: 24 }}>
        <DeviceInfoRow icon={Shield} label="OS" value={device.os || 'Unknown'} />
        <DeviceInfoRow icon={Globe} label="DNS" value={device.dns || 'Unknown'} />
        <DeviceInfoRow icon={Clock} label="First Seen" value={device.first_seen ? new Date(device.first_seen).toLocaleString() : 'N/A'} />
        <DeviceInfoRow icon={Activity} label="Last Seen" value={device.last_seen ? new Date(device.last_seen).toLocaleString() : 'N/A'} />
      </div>

      <div style={{ height: 1, background: 'rgba(255,255,255,0.1)', marginBottom: 24 }} />

      {/* Open Ports Section */}
      {hasPorts && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#bfdbfe', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Server size={20} />
            Open Ports
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.tcp_open_ports?.map((port, i) => (
              <Badge key={`tcp-${i}`} style={{
                background: 'rgba(59, 130, 246, 0.4)',
                color: '#dbeafe',
                border: '1px solid rgba(59, 130, 246, 0.6)',
                fontWeight: 700,
                fontSize: 13
              }}>
                TCP:{port}
              </Badge>
            ))}
            {device.udp_open_ports?.map((port, i) => (
              <Badge key={`udp-${i}`} style={{
                background: 'rgba(168, 85, 247, 0.4)',
                color: '#e9d5ff',
                border: '1px solid rgba(168, 85, 247, 0.6)',
                fontWeight: 700,
                fontSize: 13
              }}>
                UDP:{port}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* Packets Section */}
      {hasPackets && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#86efac', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Activity size={20} />
            ICMP Packets
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.packets_reached?.map((packets, i) => (
              <span key={i} style={{
                fontSize: 14,
                padding: '8px 14px',
                borderRadius: 6,
                background: 'rgba(34, 197, 94, 0.3)',
                color: '#bbf7d0',
                border: '1px solid rgba(34, 197, 94, 0.5)',
                fontFamily: 'monospace',
                fontWeight: 700
              }}>
                {packets}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* TCP Banner Section */}
      {hasBanner && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#fde047', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Server size={20} />
            TCP Banner
          </div>
          <pre style={{
            fontSize: 13,
            color: '#fef08a',
            margin: 0,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            padding: 12,
            background: 'rgba(234, 179, 8, 0.2)',
            borderRadius: 6,
            border: '1px solid rgba(234, 179, 8, 0.4)'
          }}>
            {device.tcp_banner}
          </pre>
        </div>
      )}

      {/* Scan Times Section */}
      {hasScanTimes && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#c7d2fe', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Clock size={20} />
            Scan Times
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.scan_times.slice(0, 10).map((time, i) => (
              <span key={i} style={{
                fontSize: 12,
                padding: '6px 10px',
                borderRadius: 4,
                background: 'rgba(99, 102, 241, 0.3)',
                color: '#e0e7ff',
                fontFamily: 'monospace',
                fontWeight: 600
              }}>
                {new Date(time).toLocaleString()}
              </span>
            ))}
            {device.scan_times.length > 10 && (
              <span style={{ fontSize: 12, color: '#c7d2fe', fontWeight: 700 }}>
                +{device.scan_times.length - 10} more...
              </span>
            )}
          </div>
        </div>
      )}

      {/* Scanner Types Section */}
      {hasScanners && (
        <div>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#fecaca', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Shield size={20} />
            Scanner Types
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.scanner_types?.map((scanner, i) => (
              <Badge key={i} style={{
                background: 'rgba(239, 68, 68, 0.4)',
                color: '#fee2e2',
                border: '1px solid rgba(239, 68, 68, 0.6)',
                fontWeight: 700,
                textTransform: 'uppercase',
                fontSize: 12
              }}>
                {scanner}
              </Badge>
            ))}
          </div>
        </div>
      )}
    </Card>
  )
}

function DeviceInfoRow({ icon: Icon, label, value }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: 12, borderRadius: 8, background: 'rgba(0, 0, 0, 0.3)' }}>
      <Icon size={16} style={{ color: '#bfdbfe' }} />
      <span style={{ fontSize: 13, color: '#bfdbfe', fontWeight: 700 }}>{label}:</span>
      <span style={{ fontSize: 14, color: '#f1f5f9', fontFamily: 'monospace', fontWeight: 700 }}>{value}</span>
    </div>
  )
}

