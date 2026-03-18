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
    return <div className="flex items-center justify-center h-64 text-gray-500">加载中...</div>
  }

  const cards = [
    { label: '总请求数', value: formatNumber(stats?.total_requests || 0), sub: `今日 ${formatNumber(stats?.requests_today || 0)}`, icon: Activity, color: 'blue' },
    { label: '总 Token 数', value: formatNumber(stats?.total_tokens || 0), sub: `今日 ${formatNumber(stats?.tokens_today || 0)}`, icon: Cpu, color: 'purple' },
    { label: '预估费用', value: formatCost(stats?.total_cost || 0), sub: '累计', icon: DollarSign, color: 'green' },
    { label: '平均延迟', value: formatLatency(stats?.avg_latency || 0), sub: `${stats?.active_providers || 0} 个活跃源`, icon: Zap, color: 'yellow' },
    { label: '成功率', value: `${(100 - (stats?.error_rate || 0)).toFixed(1)}%`, sub: `${(stats?.error_rate || 0).toFixed(1)}% 错误`, icon: TrendingUp, color: 'emerald' },
    { label: '活跃 API 源', value: String(stats?.active_providers || 0), sub: '在线', icon: AlertCircle, color: 'cyan' },
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
      <h2 className="text-2xl font-bold text-gray-900 mb-6">仪表板</h2>

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
          <h3 className="text-sm font-semibold text-gray-700 mb-4">请求趋势</h3>
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
          <h3 className="text-sm font-semibold text-gray-700 mb-4">Token 用量趋势</h3>
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
