import { useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Search, Network, Shield, Globe, Server, Clock, Activity } from 'lucide-react'
import { api }           from '@/api/http'
import { Button, Badge, EmptyState, Card, Spinner } from '@/components/ui'
import toast             from 'react-hot-toast'

export default function SearchPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const returnTo = searchParams.get('returnTo') || '/admin'

  const [results, setResults] = useState(null)
  const [loading, setLoading] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)
  const [selectedDeviceId, setSelectedDeviceId] = useState(null)
  const [query, setQuery] = useState('')

  const backButtonLabel = useMemo(() => {
    return returnTo === '/admin' ? 'Back to dashboard' : 'Back'
  }, [returnTo])

  return (
    <div className="search-page">
      <div className="search-topbar">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => navigate(returnTo)}
        >
          {backButtonLabel}
        </Button>
      </div>

      <div className={`search-hero${hasSearched ? ' search-hero--compact' : ''}`}>
        <div className="search-hero-inner">
          <NewDeviceSearch
            query={query}
            setQuery={setQuery}
            loading={loading}
            setLoading={setLoading}
            setResults={setResults}
            setHasSearched={setHasSearched}
            setSelectedDeviceId={setSelectedDeviceId}
          />

          {!hasSearched && (
            <div className="search-helper">
              Try <code>mac: xx:xx:xx:xx:xx:xx</code>, <code>ip: x.x.x.x</code>, or <code>x.x.x.x/yy</code>
            </div>
          )}
        </div>
      </div>

      {hasSearched && (
        <div className="search-results">
          {loading && <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}><Spinner size="lg" /></div>}
          {!loading && results === null && <EmptyState title="Device Search" description="Enter a MAC or IP address to search" />}
          {!loading && results !== null && !results.found && <EmptyState title="No results found" description="Try a different MAC or IP address" />}
          {!loading && results?.found && (
            <DeviceSearchResult
              data={results.data}
              selectedDeviceId={selectedDeviceId}
              setSelectedDeviceId={setSelectedDeviceId}
              onFillQuery={setQuery}
            />
          )}
        </div>
      )}
    </div>
  )
}

function NewDeviceSearch({ query, setQuery, setResults, loading, setLoading, setHasSearched, setSelectedDeviceId }) {

  const handleUniversalSearch = async () => {
    if (!query.trim()) return

    setHasSearched(true)
    setSelectedDeviceId(null)
    setLoading(true)
    setResults(null)
    try {
      const res = await api.universalSearch(query)
      if (!res.success) {
        toast.error(res.error ?? 'Search failed')
        setResults(res)
      } else {
        setResults(res)
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
    <Card className="search-card">
      <div className="search-input-row">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="mac: xx:xx:xx:xx:xx:xx or ip: x.x.x.x"
          className="search-input"
        />
        <Button
          variant="primary"
          loading={loading}
          onClick={handleUniversalSearch}
          disabled={!query.trim()}
          className="search-button"
        >
          <Search size={16} />
        </Button>
      </div>
      <div className="search-card-hint">
        Format: <code>mac: xx:xx:xx:xx:xx:xx</code>, <code>ip: x.x.x.x</code>, <code>ip: x.x.x.x/yy</code>, or <code>x.x.x.x/yy</code>
      </div>
    </Card>
  )
}

function DeviceSearchResult({ data, selectedDeviceId, setSelectedDeviceId, onFillQuery }) {
  if (!data) return null

  if (Array.isArray(data)) {
    const devices = data
    const activeDevice = devices.find((device) => device.id === selectedDeviceId)

    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <Card>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
            <div>
              <div className="card-title" style={{ marginBottom: 4 }}>
                CIDR Results
              </div>
              <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>
                {devices.length} device{devices.length === 1 ? '' : 's'} matched
              </div>
            </div>
            <Badge color="blue" dot={false}>{devices.length}</Badge>
          </div>

          <div style={{ display: 'grid', gap: 10 }}>
            {devices.map((device) => {
              const isActive = activeDevice?.id === device.id
              return (
                <div key={device.id}>
                  <button
                    type="button"
                    onClick={() => setSelectedDeviceId((current) => (current === device.id ? null : device.id))}
                    className="search-hit-row"
                    style={{
                      textAlign: 'left',
                      border: isActive ? '1px solid var(--blue)' : '1px solid var(--border)',
                      background: isActive ? 'var(--blue-dim)' : 'var(--bg-hover)',
                      color: 'inherit',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
                      <div>
                        <div style={{ fontFamily: 'var(--font-mono)', fontWeight: 700 }}>{device.id}</div>
                        <div style={{ fontSize: 12, color: 'var(--text-secondary)', marginTop: 2 }}>
                          {device.mac || 'No MAC'} • {device.os || 'Unknown OS'}
                        </div>
                      </div>
                      <Badge color={isActive ? 'blue' : 'gray'} dot={false}>{isActive ? 'Hide' : 'Open'}</Badge>
                    </div>
                  </button>

                  {isActive && (
                    <div style={{ marginTop: 12, paddingLeft: 8 }}>
                      <L3DeviceCard device={device} onFillQuery={onFillQuery} />
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </Card>

        {!activeDevice && devices.length > 0 && (
          <div style={{ textAlign: 'center', color: 'var(--text-secondary)', fontSize: 13 }}>
            Click any IP to open details here.
          </div>
        )}
      </div>
    )
  }

  // Check if it's L2 or L3 device based on structure
  // API returns 'id' field (not '_id'), so we check both
  // L3 devices have IP address in id field (contains dots) or ip field
  // L2 devices have MAC address in id field (contains colons) without IP
  const idField = data.id || data._id
  const isL3 = idField && (idField.includes('.') || data.ip !== undefined)
  const isL2 = !isL3 && idField && idField.includes(':')

  if (isL3) {
    return <L3DeviceCard device={data} onFillQuery={onFillQuery} />
  } else if (isL2) {
    return <L2DeviceCard device={data} onFillQuery={onFillQuery} />
  }

  return <EmptyState title="Unknown device type" />
}

function L2DeviceCard({ device, onFillQuery }) {
  const hasIpAddresses = device.ip_addresses?.length > 0
  const hasScanTimes = device.scan_times?.length > 0
  const hasScanners = device.scanner_types?.length > 0
  const [visibleIpCount, setVisibleIpCount] = useState(15)
  const [showAllScanTimes, setShowAllScanTimes] = useState(false)

  // Use 'id' field (MAC address) or fallback to 'mac' or '_id'
  const displayId = device.id || device.mac || device._id
  const visibleIpAddresses = device.ip_addresses?.slice(0, visibleIpCount) ?? []
  const hasMoreIpAddresses = (device.ip_addresses?.length ?? 0) > visibleIpCount
  const visibleScanTimes = showAllScanTimes ? (device.scan_times ?? []) : (device.scan_times?.slice(0, 10) ?? [])
  const hiddenScanTimesCount = Math.max((device.scan_times?.length ?? 0) - visibleScanTimes.length, 0)

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
            {visibleIpAddresses.map((ip, i) => (
              <button
                key={i}
                type="button"
                className="search-inline-link"
                onClick={() => onFillQuery(`ip: ${ip}`)}
              >
                <Badge dot={false} style={{
                  background: 'rgba(59, 130, 246, 0.3)',
                  color: '#bfdbfe',
                  border: '1px solid rgba(59, 130, 246, 0.5)',
                  fontFamily: 'monospace',
                  fontWeight: 700,
                  fontSize: 13
                }}>
                  {ip}
                </Badge>
              </button>
            ))}
          </div>
          {hasMoreIpAddresses && (
            <div style={{ marginTop: 12, display: 'flex', justifyContent: 'center' }}>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setVisibleIpCount((count) => count + 15)}
              >
                Загрузить ещё IP ({Math.min(15, device.ip_addresses.length - visibleIpCount)})
              </Button>
            </div>
          )}
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
            {visibleScanTimes.map((time, i) => (
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
          </div>
          {hiddenScanTimesCount > 0 && (
            <div style={{ marginTop: 12, display: 'flex', justifyContent: 'center' }}>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setShowAllScanTimes(true)}
              >
                Показать все времена (+{hiddenScanTimesCount})
              </Button>
            </div>
          )}
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
              <Badge key={i} dot={false} style={{
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

function L3DeviceCard({ device, onFillQuery }) {
  const hasPorts = (device.tcp_open_ports?.length > 0) || (device.udp_open_ports?.length > 0)
  const hasPackets = device.packets_reached?.length > 0
  const bannerEntries = Object.entries(device.tcp_banners ?? {}).filter(([port, banner]) => port && banner)
  const hasBannerByPort = bannerEntries.length > 0
  const hasScanTimes = device.scan_times?.length > 0
  const hasScanners = device.scanner_types?.length > 0
  const [showAllScanTimes, setShowAllScanTimes] = useState(false)
  const [expandedBannerPort, setExpandedBannerPort] = useState(null)

  // Use 'id' field (IP address) or fallback to 'ip' or '_id'
  const displayId = device.id || device.ip || device._id
  const visibleScanTimes = showAllScanTimes ? (device.scan_times ?? []) : (device.scan_times?.slice(0, 10) ?? [])
  const hiddenScanTimesCount = Math.max((device.scan_times?.length ?? 0) - visibleScanTimes.length, 0)

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
          {device.mac && device.mac !== '-' && (
            <button
              type="button"
              className="search-inline-link"
              onClick={() => onFillQuery(`mac: ${device.mac}`)}
            >
              <Badge dot={false} style={{ background: 'rgba(168, 85, 247, 0.4)', color: '#ddd6fe', fontFamily: 'monospace', fontWeight: 700, fontSize: 15, padding: '8px 12px' }}>
                {device.mac}
              </Badge>
            </button>
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
              <Badge key={`tcp-${i}`} dot={false} style={{
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
              <Badge key={`udp-${i}`} dot={false} style={{
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
      {hasBannerByPort && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: '#fde047', marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
            <Server size={20} />
            TCP Banners by Port
          </div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 10 }}>
            {bannerEntries.map(([port]) => (
              <button
                key={port}
                type="button"
                onClick={() => setExpandedBannerPort((prev) => (prev === port ? null : port))}
                style={{
                  fontSize: 12,
                  fontFamily: 'monospace',
                  fontWeight: 700,
                  padding: '6px 10px',
                  borderRadius: 6,
                  cursor: 'pointer',
                  border: expandedBannerPort === port
                    ? '1px solid rgba(250, 204, 21, 0.9)'
                    : '1px solid rgba(234, 179, 8, 0.45)',
                  background: expandedBannerPort === port
                    ? 'rgba(234, 179, 8, 0.4)'
                    : 'rgba(234, 179, 8, 0.2)',
                  color: '#fef08a'
                }}
              >
                TCP:{port}
              </button>
            ))}
          </div>
          {expandedBannerPort && (
            <div style={{
              padding: 12,
              background: 'rgba(234, 179, 8, 0.2)',
              borderRadius: 6,
              border: '1px solid rgba(234, 179, 8, 0.4)'
            }}>
              <div style={{ fontSize: 12, color: '#fde68a', marginBottom: 6, fontFamily: 'monospace', fontWeight: 700 }}>
                Port {expandedBannerPort}
              </div>
              <pre style={{
                fontSize: 13,
                color: '#fef08a',
                margin: 0,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
              }}>
                {device.tcp_banners?.[expandedBannerPort]}
              </pre>
            </div>
          )}
          {!expandedBannerPort && (
            <div style={{ fontSize: 12, color: '#fde68a' }}>
              Нажмите на порт, чтобы раскрыть баннер.
            </div>
          )}
          <div style={{ marginTop: 6, fontSize: 12, color: '#fde68a', opacity: 0.9 }}>
            Порты: {bannerEntries.map(([port]) => port).join(',')}
          </div>
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
            {visibleScanTimes.map((time, i) => (
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
          </div>
          {hiddenScanTimesCount > 0 && (
            <div style={{ marginTop: 12, display: 'flex', justifyContent: 'center' }}>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setShowAllScanTimes(true)}
              >
                Показать все времена (+{hiddenScanTimesCount})
              </Button>
            </div>
          )}
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
              <Badge key={i} dot={false} style={{
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

