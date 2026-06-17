import { useState } from 'react'
import { Search, Network, Radio, Shield, Terminal, Globe, Server, Clock, Activity } from 'lucide-react'
import { api }           from '@/api/http'
import { Button, Badge, EmptyState, Card, Spinner } from '@/components/ui'
import toast             from 'react-hot-toast'

const SCANNER_TYPES = [
  { id: 'new',  label: 'Device Search', Icon: Search,  color: 'blue'   },
  { id: 'arp',  label: 'ARP',  Icon: Network,  color: 'green'  },
  { id: 'icmp', label: 'ICMP', Icon: Radio,    color: 'blue'   },
  { id: 'nmap', label: 'Nmap', Icon: Shield,   color: 'purple' },
  { id: 'tcp',  label: 'TCP',  Icon: Terminal, color: 'yellow' },
]

export default function SearchPage() {
  const [type,    setType]    = useState('new')
  const [results, setResults] = useState(null)
  const [loading, setLoading] = useState(false)

  const selected = SCANNER_TYPES.find((t) => t.id === type)

  const handleSearch = async (body) => {
    setLoading(true)
    setResults(null)
    try {
      const res = await api.search(type, body)
      if (!res.success) {
        toast.error(res.error ?? 'Search failed')
      } else {
        setResults(res)
      }
    } catch {
      toast.error('Search failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title"><Search size={22} /> Search</h1>
          <p className="page-subtitle">Query historical scan data by target parameters</p>
        </div>
      </div>

      <Card title="Scanner Type" style={{ marginBottom: 20 }}>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          {SCANNER_TYPES.map(({ id, label, Icon, color }) => (
            <button
              key={id}
              onClick={() => { setType(id); setResults(null) }}
              className={`scanner-type-btn${type === id ? ` active color-${color}` : ''}`}
            >
              <Icon size={14} />
              {label}
            </button>
          ))}
        </div>
      </Card>

      {type === 'new' ? (
        <NewDeviceSearch onResult={setResults} loading={loading} setLoading={setLoading} />
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.5fr', gap: 20, alignItems: 'start' }}>
          <Card title={`Search ${selected?.label}`}>
            {type === 'arp'  && <ARPForm  onSearch={handleSearch} loading={loading} />}
            {type === 'icmp' && <ICMPForm onSearch={handleSearch} loading={loading} />}
            {type === 'nmap' && <NmapForm onSearch={handleSearch} loading={loading} />}
            {type === 'tcp'  && <TCPForm  onSearch={handleSearch} loading={loading} />}
          </Card>

          <Card title="Results">
            {loading && <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}><Spinner size="lg" /></div>}
            {!loading && results === null && <EmptyState title="Run a search" description="Fill in the form and press Search" />}
            {!loading && results !== null && !results.found && <EmptyState title="No results found" />}
            {!loading && results?.found && (
              <SearchResults type={type} data={results.data} count={results.count} />
            )}
          </Card>
        </div>
      )}
    </div>
  )
}

function NewDeviceSearch({ onResult, loading, setLoading }) {
  const [query, setQuery] = useState('')

  const handleUniversalSearch = async () => {
    if (!query.trim()) return

    setLoading(true)
    try {
      const res = await api.universalSearch(query)
      if (!res.success) {
        toast.error(res.error ?? 'Search failed')
        onResult(null)
      } else {
        onResult(res)
      }
    } catch {
      toast.error('Search failed')
      onResult(null)
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
              placeholder="mac:xx:xx:xx:xx:xx:xx or ip:x.x.x.x"
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
          Examples: <code style={{ background: 'var(--bg-secondary)', padding: '2px 6px', borderRadius: 4 }}>mac:00:11:22:33:44:55</code> or <code style={{ background: 'var(--bg-secondary)', padding: '2px 6px', borderRadius: 4 }}>ip:192.168.1.1</code>
        </div>
      </Card>

      {loading && <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}><Spinner size="lg" /></div>}
      {!loading && onResult === null && <EmptyState title="Device Search" description="Enter a MAC or IP address to search" />}
      {!loading && onResult !== null && !onResult.found && <EmptyState title="No results found" description="Try a different MAC or IP address" />}
      {!loading && onResult?.found && (
        <DeviceSearchResult data={onResult.data} />
      )}
    </div>
  )
}

function DeviceSearchResult({ data }) {
  if (!data) return null

  // Check if it's L2 or L3 device based on structure
  const isL2 = data.mac && !data.ip
  const isL3 = data.ip

  if (isL2) {
    return <L2DeviceCard device={data} />
  } else if (isL3) {
    return <L3DeviceCard device={data} />
  }

  return <EmptyState title="Unknown device type" />
}

function L2DeviceCard({ device }) {
  return (
    <Card style={{ 
      background: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 100%)',
      border: '1px solid rgba(99, 102, 241, 0.3)',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <div style={{
            width: 48,
            height: 48,
            borderRadius: 12,
            background: 'rgba(99, 102, 241, 0.2)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <Network size={24} style={{ color: '#818cf8' }} />
          </div>
          <div>
            <div style={{ fontSize: 18, fontWeight: 600, color: '#fff' }}>
              {device.mac || device.id}
            </div>
            <div style={{ fontSize: 12, color: '#94a3b8' }}>
              L2 Device
            </div>
          </div>
        </div>
        <Badge style={{ background: 'rgba(34, 197, 94, 0.2)', color: '#4ade80' }}>
          Active
        </Badge>
      </div>

      <div style={{ display: 'grid', gap: 12, marginBottom: 16 }}>
        <DeviceInfoRow icon={Globe} label="Vendor" value={device.vendor || 'Unknown'} />
        <DeviceInfoRow icon={Server} label="IP Addresses" value={device.ip_addresses?.join(', ') || 'None'} />
        <DeviceInfoRow icon={Clock} label="First Seen" value={new Date(device.first_seen).toLocaleString()} />
        <DeviceInfoRow icon={Activity} label="Last Seen" value={new Date(device.last_seen).toLocaleString()} />
      </div>

      <div style={{ 
        padding: 12, 
        borderRadius: 8, 
        background: 'rgba(0, 0, 0, 0.2)',
        marginBottom: 12
      }}>
        <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>Scan Information</div>
        <div style={{ display: 'grid', gap: 8 }}>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.scan_times?.slice(0, 3).map((time, i) => (
              <span key={i} style={{ 
                fontSize: 11, 
                padding: '4px 8px', 
                borderRadius: 4, 
                background: 'rgba(99, 102, 241, 0.1)',
                color: '#a5b4fc'
              }}>
                {new Date(time).toLocaleString()}
              </span>
            ))}
          </div>
          <div style={{ fontSize: 11, color: '#94a3b8' }}>
            Scanners: {device.scanner_types?.join(', ') || 'None'}
          </div>
        </div>
      </div>
    </Card>
  )
}

function L3DeviceCard({ device }) {
  return (
    <Card style={{ 
      background: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 100%)',
      border: '1px solid rgba(99, 102, 241, 0.3)',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <div style={{
            width: 48,
            height: 48,
            borderRadius: 12,
            background: 'rgba(99, 102, 241, 0.2)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <Server size={24} style={{ color: '#818cf8' }} />
          </div>
          <div>
            <div style={{ fontSize: 18, fontWeight: 600, color: '#fff' }}>
              {device.ip || device.id}
            </div>
            <div style={{ fontSize: 12, color: '#94a3b8' }}>
              L3 Device
            </div>
          </div>
        </div>
        <Badge style={{ background: 'rgba(34, 197, 94, 0.2)', color: '#4ade80' }}>
          Active
        </Badge>
      </div>

      <div style={{ display: 'grid', gap: 12, marginBottom: 16 }}>
        <DeviceInfoRow icon={Network} label="MAC Address" value={device.mac || 'N/A'} />
        <DeviceInfoRow icon={Shield} label="OS" value={device.os || 'Unknown'} />
        <DeviceInfoRow icon={Globe} label="DNS" value={device.dns || 'Unknown'} />
        <DeviceInfoRow icon={Clock} label="First Seen" value={new Date(device.first_seen).toLocaleString()} />
        <DeviceInfoRow icon={Activity} label="Last Seen" value={new Date(device.last_seen).toLocaleString()} />
      </div>

      {(device.tcp_open_ports?.length > 0 || device.udp_open_ports?.length > 0) && (
        <div style={{ 
          padding: 12, 
          borderRadius: 8, 
          background: 'rgba(0, 0, 0, 0.2)',
          marginBottom: 12
        }}>
          <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>Open Ports</div>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.tcp_open_ports?.map((port, i) => (
              <Badge key={`tcp-${i}`} style={{ background: 'rgba(59, 130, 246, 0.2)', color: '#60a5fa' }}>
                TCP:{port}
              </Badge>
            ))}
            {device.udp_open_ports?.map((port, i) => (
              <Badge key={`udp-${i}`} style={{ background: 'rgba(168, 85, 247, 0.2)', color: '#a78bfa' }}>
                UDP:{port}
              </Badge>
            ))}
          </div>
        </div>
      )}

      <div style={{ 
        padding: 12, 
        borderRadius: 8, 
        background: 'rgba(0, 0, 0, 0.2)',
        marginBottom: 12
      }}>
        <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>Scan Information</div>
        <div style={{ display: 'grid', gap: 8 }}>
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
            {device.scan_times?.slice(0, 3).map((time, i) => (
              <span key={i} style={{ 
                fontSize: 11, 
                padding: '4px 8px', 
                borderRadius: 4, 
                background: 'rgba(99, 102, 241, 0.1)',
                color: '#a5b4fc'
              }}>
                {new Date(time).toLocaleString()}
              </span>
            ))}
          </div>
          <div style={{ fontSize: 11, color: '#94a3b8' }}>
            Scanners: {device.scanner_types?.join(', ') || 'None'}
          </div>
        </div>
      </div>

      {device.tcp_banner && device.tcp_banner !== '-' && (
        <div style={{ 
          padding: 12, 
          borderRadius: 8, 
          background: 'rgba(0, 0, 0, 0.2)'
        }}>
          <div style={{ fontSize: 12, color: '#94a3b8', marginBottom: 8 }}>TCP Banner</div>
          <pre style={{ 
            fontSize: 11, 
            color: '#e2e8f0', 
            margin: 0, 
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word'
          }}>
            {device.tcp_banner}
          </pre>
        </div>
      )}
    </Card>
  )
}

function DeviceInfoRow({ icon: Icon, label, value }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <Icon size={16} style={{ color: '#94a3b8' }} />
      <span style={{ fontSize: 13, color: '#94a3b8' }}>{label}:</span>
      <span style={{ fontSize: 13, color: '#e2e8f0', fontFamily: 'monospace' }}>{value}</span>
    </div>
  )
}

function ARPForm({ onSearch, loading }) {
  const [iface,   setIface]   = useState('')
  const [ipRange, setIpRange] = useState('')
  return (
    <div style={{ display: 'grid', gap: 12 }}>
      <div className="form-group">
        <label>Interface Name</label>
        <input value={iface} onChange={(e) => setIface(e.target.value)} placeholder="eth0" />
      </div>
      <div className="form-group">
        <label>IP Range (CIDR)</label>
        <input value={ipRange} onChange={(e) => setIpRange(e.target.value)} placeholder="192.168.1.0/24" />
      </div>
      <Button variant="primary" loading={loading} onClick={() => onSearch({ interface_name: iface, ip_range: ipRange })} disabled={!iface && !ipRange}>
        Search
      </Button>
    </div>
  )
}

function ICMPForm({ onSearch, loading }) {
  const [targets, setTargets] = useState('')
  const parsed = targets.split(/[\n,]+/).map((t) => t.trim()).filter(Boolean)
  return (
    <div style={{ display: 'grid', gap: 12 }}>
      <div className="form-group">
        <label>Targets (comma or newline)</label>
        <textarea rows={3} value={targets} onChange={(e) => setTargets(e.target.value)} placeholder="192.168.1.1, google.com" />
      </div>
      <Button variant="primary" loading={loading} onClick={() => onSearch({ targets: parsed })} disabled={!parsed.length}>
        Search
      </Button>
    </div>
  )
}

function NmapForm({ onSearch, loading }) {
  const [ip,     setIp]     = useState('')
  const [method, setMethod] = useState('tcp_udp_scan')
  return (
    <div style={{ display: 'grid', gap: 12 }}>
      <div className="form-group">
        <label>IP Address</label>
        <input value={ip} onChange={(e) => setIp(e.target.value)} placeholder="192.168.1.1" />
      </div>
      <div className="form-group">
        <label>Scan Method</label>
        <select value={method} onChange={(e) => setMethod(e.target.value)}>
          <option value="tcp_udp_scan">TCP / UDP Scan</option>
          <option value="os_detection">OS Detection</option>
          <option value="host_discovery">Host Discovery</option>
        </select>
      </div>
      <Button variant="primary" loading={loading} onClick={() => onSearch({ ip, scan_method: method })} disabled={!ip}>
        Search
      </Button>
    </div>
  )
}

function TCPForm({ onSearch, loading }) {
  const [host, setHost] = useState('')
  const [port, setPort] = useState('')
  return (
    <div style={{ display: 'grid', gap: 12 }}>
      <div className="form-group">
        <label>Host</label>
        <input value={host} onChange={(e) => setHost(e.target.value)} placeholder="192.168.1.1" />
      </div>
      <div className="form-group">
        <label>Port</label>
        <input value={port} onChange={(e) => setPort(e.target.value)} placeholder="80" />
      </div>
      <Button variant="primary" loading={loading} onClick={() => onSearch({ host, port })} disabled={!host && !port}>
        Search
      </Button>
    </div>
  )
}

function SearchResults({ type, data, count }) {
  const rows = Array.isArray(data) ? data : []
  return (
    <div>
      <div style={{ marginBottom: 12, fontSize: 12, color: 'var(--text-muted)' }}>
        Found <strong style={{ color: 'var(--text-primary)' }}>{count}</strong> record{count !== 1 ? 's' : ''}
      </div>
      {type === 'arp'  && <ARPResults  rows={rows} />}
      {type === 'icmp' && <ICMPResults rows={rows} />}
      {type === 'nmap' && <NmapResults rows={rows} />}
      {type === 'tcp'  && <TCPResults  rows={rows} />}
    </div>
  )
}

function ARPResults({ rows }) {
  return rows.map((r, i) => (
    <div key={i} style={{ marginBottom: 16 }}>
      <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
        <Badge>{r.status}</Badge>
        <span className="td-mono text-sm">{r.ip_range} via {r.interface_name}</span>
        <span className="text-muted text-sm">{r.online_count}/{r.total_count} online</span>
      </div>
      <div className="table-wrap">
        <table>
          <thead><tr><th>IP</th><th>MAC</th><th>Status</th></tr></thead>
          <tbody>
            {(r.devices ?? []).slice(0, 10).map((d, j) => (
              <tr key={j}><td className="td-mono">{d.ip}</td><td className="td-mono">{d.mac}</td><td><Badge>{d.status}</Badge></td></tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  ))
}

function ICMPResults({ rows }) {
  return rows.map((r, i) => (
    <div key={i} style={{ marginBottom: 16 }}>
      <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
        <Badge>{r.status}</Badge>
        <span className="td-mono text-sm">{(r.targets ?? []).join(', ')}</span>
      </div>
      <div className="table-wrap">
        <table>
          <thead><tr><th>Target</th><th>Sent</th><th>Recv</th><th>Loss</th></tr></thead>
          <tbody>
            {(r.results ?? []).map((res, j) => (
              <tr key={j}>
                <td className="td-mono">{res.target}</td>
                <td className="td-mono" style={{ textAlign: 'right' }}>{res.packets_sent}</td>
                <td className="td-mono" style={{ textAlign: 'right' }}>{res.packets_received}</td>
                <td className="td-mono" style={{ textAlign: 'right', color: res.packet_loss_percent > 0 ? 'var(--red)' : 'var(--green)' }}>
                  {res.packet_loss_percent?.toFixed(1)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  ))
}

function NmapResults({ rows }) {
  return rows.map((r, i) => (
    <div key={i} style={{ marginBottom: 12 }}>
      <div style={{ display: 'flex', gap: 8, marginBottom: 6 }}>
        <Badge>{r.status}</Badge>
        <span className="td-mono text-sm">{r.ip || r.host}</span>
        {r.name && <span className="text-muted text-sm">{r.name}</span>}
      </div>
    </div>
  ))
}

function TCPResults({ rows }) {
  return rows.map((r, i) => (
    <div key={i} style={{ marginBottom: 16 }}>
      <div style={{ display: 'flex', gap: 8, marginBottom: 6 }}>
        <Badge>{r.status}</Badge>
        <span className="td-mono text-sm">{r.host}:{r.port}</span>
      </div>
      {r.decoded_text && <pre className="decoded-text" style={{ maxHeight: 150 }}>{r.decoded_text}</pre>}
    </div>
  ))
}

