import { useState, useEffect } from 'react'
import { Activity, Cpu, DollarSign, Zap, TrendingUp, AlertCircle } from 'lucide-react'
import { getDashboard, getTimeSeries } from '@/lib/api'
import { formatNumber, formatCost, formatLatency } from '@/lib/utils'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts'

export default function DashboardPage() {
  const [stats, setStats] = useState<any>(null)
  const [timeSeries, setTimeSeries] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [dashData, tsData] = await Promise.all([
        getDashboard(),
        getTimeSeries({ period: 'hour', days: 7 }),
      ])
      setStats(dashData)
      setTimeSeries(tsData)
    } catch (err) {
      console.error('Failed to load dashboard:', err)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div className="flex items-center justify-center h-64 text-gray-500">Loading...</div>
  }

  const cards = [
    { label: 'Total Requests', value: formatNumber(stats?.total_requests || 0), sub: `${formatNumber(stats?.requests_today || 0)} today`, icon: Activity, color: 'blue' },
    { label: 'Total Tokens', value: formatNumber(stats?.total_tokens || 0), sub: `${formatNumber(stats?.tokens_today || 0)} today`, icon: Cpu, color: 'purple' },
    { label: 'Estimated Cost', value: formatCost(stats?.total_cost || 0), sub: 'All time', icon: DollarSign, color: 'green' },
    { label: 'Avg Latency', value: formatLatency(stats?.avg_latency || 0), sub: `${stats?.active_providers || 0} active providers`, icon: Zap, color: 'yellow' },
    { label: 'Success Rate', value: `${(100 - (stats?.error_rate || 0)).toFixed(1)}%`, sub: `${(stats?.error_rate || 0).toFixed(1)}% errors`, icon: TrendingUp, color: 'emerald' },
    { label: 'Active Providers', value: String(stats?.active_providers || 0), sub: 'Online', icon: AlertCircle, color: 'cyan' },
  ]

  const colorMap: Record<string, string> = {
    blue: 'bg-blue-50 text-blue-600',
    purple: 'bg-purple-50 text-purple-600',
    green: 'bg-green-50 text-green-600',
    yellow: 'bg-yellow-50 text-yellow-600',
    emerald: 'bg-emerald-50 text-emerald-600',
    cyan: 'bg-cyan-50 text-cyan-600',
  }

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Dashboard</h2>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
        {cards.map((card) => (
          <div key={card.label} className="bg-white rounded-xl border border-gray-200 p-5">
            <div className="flex items-center justify-between mb-3">
              <span className="text-sm font-medium text-gray-500">{card.label}</span>
              <div className={`p-2 rounded-lg ${colorMap[card.color]}`}>
                <card.icon size={16} />
              </div>
            </div>
            <div className="text-2xl font-bold text-gray-900">{card.value}</div>
            <div className="text-xs text-gray-500 mt-1">{card.sub}</div>
          </div>
        ))}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl border border-gray-200 p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-4">Requests Over Time</h3>
          <ResponsiveContainer width="100%" height={250}>
            <AreaChart data={timeSeries}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="time" tick={{ fontSize: 11 }} tickFormatter={(v) => v.slice(11, 16) || v.slice(5, 10)} />
              <YAxis tick={{ fontSize: 11 }} />
              <Tooltip />
              <Area type="monotone" dataKey="requests" stroke="#3b82f6" fill="#dbeafe" />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-4">Token Usage Over Time</h3>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={timeSeries}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="time" tick={{ fontSize: 11 }} tickFormatter={(v) => v.slice(11, 16) || v.slice(5, 10)} />
              <YAxis tick={{ fontSize: 11 }} />
              <Tooltip />
              <Line type="monotone" dataKey="tokens" stroke="#8b5cf6" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  )
}
