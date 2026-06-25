import { create } from 'zustand'

// Load auto scan state from localStorage
const loadAutoScanState = (key) => {
  try {
    const saved = localStorage.getItem(key)
    return saved === 'true'
  } catch {
    return false
  }
}

export const useStore = create((set, get) => ({
  wsStatus:      'disconnected',
  activeScan:    null,
  latestResult:  null,
  recentResults: [],

  // ── Auto scan state (persisted in localStorage) ───────────────────────────
  arpAutoScanRunning: loadAutoScanState('arpAutoScanRunning'),
  icmpAutoScanRunning: loadAutoScanState('icmpAutoScanRunning'),

  // ── Change Detection (populated by WS change_event messages) ─────────────
  changeEvents:     [],
  newChangesCount:  0,

  setWsStatus: (wsStatus) => set({ wsStatus }),

  startScan: (scanner_service, options) =>
    set({ activeScan: { scanner_service, options, startedAt: Date.now() }, latestResult: null }),

  finishScan: (response, broadcastScannerService) => {
    const active = get().activeScan
    const entry  = {
      ...response,
      scanner_service: broadcastScannerService || active?.scanner_service || 'unknown',
      options:         active?.options ?? {},
      receivedAt:      Date.now(),
    }
    set((s) => ({
      activeScan:    null,
      latestResult:  entry,
      recentResults: [entry, ...s.recentResults].slice(0, 100),
    }))
  },

  clearRecent: () => set({ recentResults: [] }),

  // ── Auto scan actions ─────────────────────────────────────────────────────
  setArpAutoScanRunning: (running) => {
    try {
      localStorage.setItem('arpAutoScanRunning', running)
    } catch (e) {
      console.warn('Failed to save auto scan state to localStorage:', e)
    }
    set({ arpAutoScanRunning: running })
  },

  setIcmpAutoScanRunning: (running) => {
    try {
      localStorage.setItem('icmpAutoScanRunning', running)
    } catch (e) {
      console.warn('Failed to save auto scan state to localStorage:', e)
    }
    set({ icmpAutoScanRunning: running })
  },

  // ── Change event actions ──────────────────────────────────────────────────
  addChangeEvent: (event) =>
    set((s) => ({
      changeEvents:    [event, ...s.changeEvents].slice(0, 500),
      newChangesCount: s.newChangesCount + 1,
    })),

  clearNewChangesCount: () => set({ newChangesCount: 0 }),

  clearAllChangeEvents: () => set({ changeEvents: [], newChangesCount: 0 }),
}))
