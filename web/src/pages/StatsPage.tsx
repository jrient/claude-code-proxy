import { useState, useEffect } from 'react'
import { getTimeSeries, getModelStats, getRecentLogs } from '@/lib/api'
import { formatNumber, formatCost, formatLatency } from '@/lib/utils'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, LineChart, Line } from 'recharts'

const COLORS = ['#3b82f6', '#8b5cf6', '#10b981', '#f59e0b', '#ef4444', '#06b6d4']

export default function StatsPage() {
  const [period, setPeriod] = useState('hour')
  const [days, setDays] = useState(7)
  const [timeSeries, setTimeSeries] = useState<any[]>([])
  const [modelStats, setModelStats] = useState<any[]>([])
  const [logs, setLogs] = useState<any[]>([])
  const [totalLogs, setTotalLogs] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => { loadData() }, [period, days])

  const loadData = async () => {
    setLoading(true)
    try {
      const [ts, ms, lg] = await Promise.all([
        getTimeSeries({ period, days }),
        getModelStats(days),
        getRecentLogs(50, 0),
      ])
      setTimeSeries(ts)
      setModelStats(ms)
      setLogs(lg.logs)
      setTotalLogs(lg.total)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Statistics</h2>
        <div className="flex items-center gap-2">
          <select value={period} onChange={e => setPeriod(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
            <option value="hour">Hourly</option>
            <option value="day">Daily</option>
          </select>
          <select value={days} onChange={e => setDays(+e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
            <option value={1}>Last 24h</option>
            <option value={7}>Last 7 days</option>
            <option value={30}>Last 30 days</option>
          </select>
        </div>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64 text-gray-500">Loading...</div>
      ) : (
        <>
          {/* Charts Row */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            <div className="bg-white rounded-xl border border-gray-200 p-5">
              <h3 className="text-sm font-semibold text-gray-700 mb-4">Requests</h3>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={timeSeries}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis dataKey="time" tick={{ fontSize: 11 }} tickFormatter={(v) => v.slice(11, 16) || v.slice(5, 10)} />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip />
                  <Bar dataKey="requests" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>

            <div className="bg-white rounded-xl border border-gray-200 p-5">
              <h3 className="text-sm font-semibold text-gray-700 mb-4">Errors</h3>
              <ResponsiveContainer width="100%" height={250}>
                <LineChart data={timeSeries}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis dataKey="time" tick={{ fontSize: 11 }} tickFormatter={(v) => v.slice(11, 16) || v.slice(5, 10)} />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip />
                  <Line type="monotone" dataKey="errors" stroke="#ef4444" strokeWidth={2} dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>

          {/* Model Stats */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            <div className="bg-white rounded-xl border border-gray-200 p-5">
              <h3 className="text-sm font-semibold text-gray-700 mb-4">Usage by Model</h3>
              {modelStats.length > 0 ? (
                <ResponsiveContainer width="100%" height={250}>
                  <PieChart>
                    <Pie data={modelStats} dataKey="requests" nameKey="model" cx="50%" cy="50%" outerRadius={90} label={({name, percent}) => `${name?.split('-')[1] || name} ${(percent*100).toFixed(0)}%`}>
                      {modelStats.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                    </Pie>
                    <Tooltip />
                  </PieChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex items-center justify-center h-[250px] text-gray-400 text-sm">No data</div>
              )}
            </div>

            <div className="bg-white rounded-xl border border-gray-200 p-5">
              <h3 className="text-sm font-semibold text-gray-700 mb-4">Model Details</h3>
              <div className="overflow-auto max-h-[250px]">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-left text-xs text-gray-500 uppercase">
                      <th className="pb-2">Model</th>
                      <th className="pb-2 text-right">Requests</th>
                      <th className="pb-2 text-right">Tokens</th>
                      <th className="pb-2 text-right">Cost</th>
                      <th className="pb-2 text-right">Latency</th>
                    </tr>
                  </thead>
                  <tbody>
                    {modelStats.map((m, i) => (
                      <tr key={i} className="border-t border-gray-100">
                        <td className="py-2 font-medium text-gray-700">{m.model || 'unknown'}</td>
                        <td className="py-2 text-right text-gray-600">{formatNumber(m.requests)}</td>
                        <td className="py-2 text-right text-gray-600">{formatNumber(m.prompt_tokens + m.completion_tokens)}</td>
                        <td className="py-2 text-right text-gray-600">{formatCost(m.estimated_cost)}</td>
                        <td className="py-2 text-right text-gray-600">{formatLatency(m.avg_latency)}</td>
                      </tr>
                    ))}
                    {modelStats.length === 0 && <tr><td colSpan={5} className="py-8 text-center text-gray-400">No data</td></tr>}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          {/* Recent Logs */}
          <div className="bg-white rounded-xl border border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-gray-700">Recent Requests</h3>
              <span className="text-xs text-gray-500">{totalLogs} total</span>
            </div>
            <div className="overflow-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-xs text-gray-500 uppercase border-b border-gray-200">
                    <th className="pb-2 px-2">Model</th>
                    <th className="pb-2 px-2 text-right">Input</th>
                    <th className="pb-2 px-2 text-right">Output</th>
                    <th className="pb-2 px-2 text-right">Latency</th>
                    <th className="pb-2 px-2 text-center">Status</th>
                    <th className="pb-2 px-2 text-center">Stream</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((l) => (
                    <tr key={l.id} className="border-t border-gray-50 hover:bg-gray-50">
                      <td className="py-2 px-2 font-medium text-gray-700">{l.model || '-'}</td>
                      <td className="py-2 px-2 text-right text-gray-600">{formatNumber(l.prompt_tokens)}</td>
                      <td className="py-2 px-2 text-right text-gray-600">{formatNumber(l.completion_tokens)}</td>
                      <td className="py-2 px-2 text-right text-gray-600">{formatLatency(l.latency_ms)}</td>
                      <td className="py-2 px-2 text-center">
                        <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${l.status_code < 400 ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                          {l.status_code || '-'}
                        </span>
                      </td>
                      <td className="py-2 px-2 text-center text-gray-500">{l.stream ? 'SSE' : 'Sync'}</td>
                    </tr>
                  ))}
                  {logs.length === 0 && <tr><td colSpan={6} className="py-8 text-center text-gray-400">No requests yet</td></tr>}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
