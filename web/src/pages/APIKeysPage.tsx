import { useState, useEffect } from 'react'
import { Plus, Trash2, Copy, Check } from 'lucide-react'
import { getAPIKeys, createAPIKey, updateAPIKey, deleteAPIKey } from '@/lib/api'
import { formatDate } from '@/lib/utils'

interface APIKeyInfo {
  id: number
  name: string
  key_prefix: string
  enabled: boolean
  rate_limit: number
  daily_token_limit: number
  allowed_models: string
  created_at: string
}

export default function APIKeysPage() {
  const [keys, setKeys] = useState<APIKeyInfo[]>([])
  const [showForm, setShowForm] = useState(false)
  const [newKey, setNewKey] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [form, setForm] = useState({ name: '', rate_limit: 60, daily_token_limit: 0, allowed_models: '' })
  const [loading, setLoading] = useState(true)

  useEffect(() => { loadKeys() }, [])

  const loadKeys = async () => {
    try {
      setKeys(await getAPIKeys())
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const res = await createAPIKey(form)
      setNewKey(res.key)
      setShowForm(false)
      setForm({ name: '', rate_limit: 60, daily_token_limit: 0, allowed_models: '' })
      loadKeys()
    } catch (err) {
      console.error(err)
    }
  }

  const handleToggle = async (k: APIKeyInfo) => {
    await updateAPIKey(k.id, { ...k, enabled: !k.enabled })
    loadKeys()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this API key?')) return
    await deleteAPIKey(id)
    loadKeys()
  }

  const handleCopy = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  if (loading) return <div className="flex items-center justify-center h-64 text-gray-500">Loading...</div>

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">API Keys</h2>
        <button onClick={() => setShowForm(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors">
          <Plus size={16} /> Create Key
        </button>
      </div>

      {/* New Key Display */}
      {newKey && (
        <div className="bg-green-50 border border-green-200 rounded-xl p-4 mb-6">
          <p className="text-sm font-medium text-green-800 mb-2">New API key created! Copy it now — it won't be shown again.</p>
          <div className="flex items-center gap-2">
            <code className="flex-1 bg-white px-3 py-2 rounded-lg text-sm font-mono border border-green-200">{newKey}</code>
            <button onClick={handleCopy} className="p-2 bg-green-600 text-white rounded-lg hover:bg-green-700">
              {copied ? <Check size={16} /> : <Copy size={16} />}
            </button>
          </div>
          <button onClick={() => setNewKey(null)} className="text-xs text-green-600 mt-2 hover:underline">Dismiss</button>
        </div>
      )}

      {/* Create Form Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl p-6 w-full max-w-lg">
            <h3 className="text-lg font-semibold mb-4">Create API Key</h3>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input value={form.name} onChange={e => setForm({...form, name: e.target.value})}
                  placeholder="e.g., Development, Production"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" required />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Rate Limit (requests/min)</label>
                <input type="number" value={form.rate_limit} onChange={e => setForm({...form, rate_limit: +e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" min={0} />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Daily Token Limit (0 = unlimited)</label>
                <input type="number" value={form.daily_token_limit} onChange={e => setForm({...form, daily_token_limit: +e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" min={0} />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Allowed Models (comma-separated, empty = all)</label>
                <input value={form.allowed_models} onChange={e => setForm({...form, allowed_models: e.target.value})}
                  placeholder="claude-sonnet-4-20250514, claude-haiku-3-5-20241022"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowForm(false)}
                  className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50">Cancel</button>
                <button type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700">Create</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Keys Table */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Name</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Key</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Rate Limit</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Token Limit</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Status</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Created</th>
              <th className="text-right px-4 py-3 text-xs font-semibold text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody>
            {keys.length === 0 ? (
              <tr><td colSpan={7} className="text-center py-12 text-gray-500">No API keys created</td></tr>
            ) : keys.map((k) => (
              <tr key={k.id} className="border-b border-gray-100 hover:bg-gray-50">
                <td className="px-4 py-3 text-sm font-medium text-gray-900">{k.name}</td>
                <td className="px-4 py-3 text-sm text-gray-500 font-mono">{k.key_prefix}</td>
                <td className="px-4 py-3 text-sm text-gray-600 text-center">{k.rate_limit}/min</td>
                <td className="px-4 py-3 text-sm text-gray-600 text-center">{k.daily_token_limit || 'Unlimited'}</td>
                <td className="px-4 py-3 text-center">
                  <button onClick={() => handleToggle(k)}
                    className={`px-2 py-0.5 rounded text-xs font-medium ${k.enabled ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                    {k.enabled ? 'Active' : 'Disabled'}
                  </button>
                </td>
                <td className="px-4 py-3 text-sm text-gray-500">{formatDate(k.created_at)}</td>
                <td className="px-4 py-3 text-right">
                  <button onClick={() => handleDelete(k.id)} className="p-1 text-gray-400 hover:text-red-600"><Trash2 size={15} /></button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
