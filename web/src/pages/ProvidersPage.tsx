import { useState, useEffect } from 'react'
import { Plus, Pencil, Trash2, CheckCircle, XCircle, AlertCircle } from 'lucide-react'
import { getProviders, createProvider, updateProvider, deleteProvider } from '@/lib/api'

interface Provider {
  id: number
  name: string
  type: string
  base_url: string
  api_key: string
  priority: number
  weight: number
  enabled: boolean
  health_status: string
  created_at: string
}

const emptyForm = { name: '', type: 'openai', base_url: '', api_key: '', priority: 1, weight: 10 }

export default function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [loading, setLoading] = useState(true)

  useEffect(() => { loadProviders() }, [])

  const loadProviders = async () => {
    try {
      const data = await getProviders()
      setProviders(data)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      if (editingId) {
        await updateProvider(editingId, { ...form, enabled: true })
      } else {
        await createProvider(form)
      }
      setShowForm(false)
      setEditingId(null)
      setForm(emptyForm)
      loadProviders()
    } catch (err) {
      console.error(err)
    }
  }

  const handleEdit = (p: Provider) => {
    setForm({ name: p.name, type: p.type, base_url: p.base_url, api_key: '', priority: p.priority, weight: p.weight })
    setEditingId(p.id)
    setShowForm(true)
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this provider?')) return
    await deleteProvider(id)
    loadProviders()
  }

  const handleToggle = async (p: Provider) => {
    await updateProvider(p.id, { ...p, enabled: !p.enabled })
    loadProviders()
  }

  const healthIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircle size={16} className="text-green-500" />
      case 'unhealthy': return <XCircle size={16} className="text-red-500" />
      default: return <AlertCircle size={16} className="text-gray-400" />
    }
  }

  if (loading) return <div className="flex items-center justify-center h-64 text-gray-500">Loading...</div>

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Providers</h2>
        <button
          onClick={() => { setForm(emptyForm); setEditingId(null); setShowForm(true) }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          <Plus size={16} /> Add Provider
        </button>
      </div>

      {/* Form Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl p-6 w-full max-w-lg">
            <h3 className="text-lg font-semibold mb-4">{editingId ? 'Edit' : 'Add'} Provider</h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input value={form.name} onChange={e => setForm({...form, name: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" required />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
                <select value={form.type} onChange={e => setForm({...form, type: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
                  <option value="openai">OpenAI Compatible</option>
                  <option value="anthropic">Anthropic Native</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Base URL</label>
                <input value={form.base_url} onChange={e => setForm({...form, base_url: e.target.value})}
                  placeholder="https://api.example.com/v1" className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" required />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">API Key</label>
                <input value={form.api_key} onChange={e => setForm({...form, api_key: e.target.value})}
                  type="password" placeholder={editingId ? '(unchanged if empty)' : ''}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Priority (lower = higher)</label>
                  <input type="number" value={form.priority} onChange={e => setForm({...form, priority: +e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" min={1} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Weight</label>
                  <input type="number" value={form.weight} onChange={e => setForm({...form, weight: +e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" min={1} />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowForm(false)}
                  className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50">Cancel</button>
                <button type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Provider Table */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Name</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Type</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Base URL</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Priority</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Weight</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Health</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Status</th>
              <th className="text-right px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody>
            {providers.length === 0 ? (
              <tr><td colSpan={8} className="text-center py-12 text-gray-500">No providers configured</td></tr>
            ) : providers.map((p) => (
              <tr key={p.id} className="border-b border-gray-100 hover:bg-gray-50">
                <td className="px-4 py-3 text-sm font-medium text-gray-900">{p.name}</td>
                <td className="px-4 py-3 text-sm text-gray-600">
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${p.type === 'anthropic' ? 'bg-orange-50 text-orange-700' : 'bg-blue-50 text-blue-700'}`}>
                    {p.type}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm text-gray-500 max-w-[200px] truncate">{p.base_url}</td>
                <td className="px-4 py-3 text-sm text-gray-600 text-center">{p.priority}</td>
                <td className="px-4 py-3 text-sm text-gray-600 text-center">{p.weight}</td>
                <td className="px-4 py-3 text-center">{healthIcon(p.health_status)}</td>
                <td className="px-4 py-3 text-center">
                  <button onClick={() => handleToggle(p)}
                    className={`px-2 py-0.5 rounded text-xs font-medium ${p.enabled ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                    {p.enabled ? 'Enabled' : 'Disabled'}
                  </button>
                </td>
                <td className="px-4 py-3 text-right">
                  <button onClick={() => handleEdit(p)} className="p-1 text-gray-400 hover:text-blue-600"><Pencil size={15} /></button>
                  <button onClick={() => handleDelete(p.id)} className="p-1 text-gray-400 hover:text-red-600 ml-1"><Trash2 size={15} /></button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
