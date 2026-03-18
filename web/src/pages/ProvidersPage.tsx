import { useState, useEffect } from 'react'
import { Plus, Pencil, Trash2, CheckCircle, XCircle, AlertCircle, GitBranch, X } from 'lucide-react'
import { getProviders, createProvider, updateProvider, deleteProvider, getModelMappings, createModelMapping, deleteModelMapping } from '@/lib/api'

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

interface ModelMapping {
  id: number
  source: string
  target: string
}

const emptyForm = { name: '', type: 'openai', base_url: '', api_key: '', priority: 1, weight: 10 }

export default function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [loading, setLoading] = useState(true)

  // Model mapping modal state
  const [mappingProvider, setMappingProvider] = useState<Provider | null>(null)
  const [mappings, setMappings] = useState<ModelMapping[]>([])
  const [mappingLoading, setMappingLoading] = useState(false)
  const [newSource, setNewSource] = useState('')
  const [newTarget, setNewTarget] = useState('')
  const [mappingError, setMappingError] = useState('')

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
    if (!confirm('确定删除此 API 源？')) return
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

  // --- Model Mapping handlers ---
  const openMappingModal = async (p: Provider) => {
    setMappingProvider(p)
    setNewSource('')
    setNewTarget('')
    setMappingError('')
    setMappingLoading(true)
    try {
      const data = await getModelMappings(p.id)
      setMappings(data)
    } catch (err) {
      console.error(err)
    } finally {
      setMappingLoading(false)
    }
  }

  const closeMappingModal = () => {
    setMappingProvider(null)
    setMappings([])
    setNewSource('')
    setNewTarget('')
    setMappingError('')
  }

  const handleAddMapping = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!mappingProvider) return
    setMappingError('')
    try {
      await createModelMapping(mappingProvider.id, newSource.trim(), newTarget.trim())
      setNewSource('')
      setNewTarget('')
      const data = await getModelMappings(mappingProvider.id)
      setMappings(data)
    } catch (err: any) {
      setMappingError(err.message || '添加失败')
    }
  }

  const handleDeleteMapping = async (mappingId: number) => {
    if (!mappingProvider) return
    try {
      await deleteModelMapping(mappingProvider.id, mappingId)
      setMappings(mappings.filter(m => m.id !== mappingId))
    } catch (err) {
      console.error(err)
    }
  }

  if (loading) return <div className="flex items-center justify-center h-64 text-gray-500">加载中...</div>

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">API 源管理</h2>
        <button
          onClick={() => { setForm(emptyForm); setEditingId(null); setShowForm(true) }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          <Plus size={16} /> 添加 API 源
        </button>
      </div>

      {/* Provider Form Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl p-6 w-full max-w-lg">
            <h3 className="text-lg font-semibold mb-4">{editingId ? '编辑' : '添加'} API 源</h3>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">名称</label>
                <input value={form.name} onChange={e => setForm({...form, name: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" required />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">类型</label>
                <select value={form.type} onChange={e => setForm({...form, type: e.target.value})}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
                  <option value="openai">OpenAI 兼容</option>
                  <option value="anthropic">Anthropic 原生</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Base URL 地址</label>
                <input value={form.base_url} onChange={e => setForm({...form, base_url: e.target.value})}
                  placeholder="https://api.example.com/v1" className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" required />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">API 密钥</label>
                <input value={form.api_key} onChange={e => setForm({...form, api_key: e.target.value})}
                  type="password" placeholder={editingId ? '(留空则不修改)' : ''}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">优先级（数字越小越优先）</label>
                  <input type="number" value={form.priority} onChange={e => setForm({...form, priority: +e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" min={1} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">权重</label>
                  <input type="number" value={form.weight} onChange={e => setForm({...form, weight: +e.target.value})}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" min={1} />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowForm(false)}
                  className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
                <button type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700">保存</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Model Mapping Modal */}
      {mappingProvider && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-xl p-6 w-full max-w-xl">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">模型映射 — <span className="text-blue-600">{mappingProvider.name}</span></h3>
              <button onClick={closeMappingModal} className="p-1 text-gray-400 hover:text-gray-700">
                <X size={18} />
              </button>
            </div>
            <p className="text-xs text-gray-500 mb-4">将客户端发送的模型名称映射到该 API 源实际支持的模型名称。</p>

            {/* Add new mapping form */}
            <form onSubmit={handleAddMapping} className="flex gap-2 mb-4">
              <input
                value={newSource}
                onChange={e => setNewSource(e.target.value)}
                placeholder="来源模型（如 claude-sonnet-4-20250514）"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              />
              <input
                value={newTarget}
                onChange={e => setNewTarget(e.target.value)}
                placeholder="目标模型（如 gpt-4o）"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                required
              />
              <button type="submit"
                className="px-3 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 whitespace-nowrap">
                添加
              </button>
            </form>
            {mappingError && <p className="text-red-500 text-xs mb-3">{mappingError}</p>}

            {/* Mapping list */}
            {mappingLoading ? (
              <div className="text-center py-6 text-gray-500 text-sm">加载中...</div>
            ) : mappings.length === 0 ? (
              <div className="text-center py-6 text-gray-400 text-sm">暂无模型映射</div>
            ) : (
              <div className="border border-gray-200 rounded-lg overflow-hidden">
                <table className="w-full">
                  <thead>
                    <tr className="bg-gray-50 border-b border-gray-200">
                      <th className="text-left px-4 py-2 text-xs font-semibold text-gray-500">来源模型</th>
                      <th className="text-left px-4 py-2 text-xs font-semibold text-gray-500">目标模型</th>
                      <th className="text-right px-4 py-2 text-xs font-semibold text-gray-500">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {mappings.map(m => (
                      <tr key={m.id} className="border-b border-gray-100 last:border-0">
                        <td className="px-4 py-2 text-sm text-gray-700 font-mono">{m.source}</td>
                        <td className="px-4 py-2 text-sm text-gray-700 font-mono">{m.target}</td>
                        <td className="px-4 py-2 text-right">
                          <button onClick={() => handleDeleteMapping(m.id)}
                            className="p-1 text-gray-400 hover:text-red-600">
                            <Trash2 size={14} />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <div className="flex justify-end mt-4">
              <button onClick={closeMappingModal}
                className="px-4 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50">
                关闭
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Provider Table */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500">名称</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500">类型</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-gray-500">Base URL</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500">优先级</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500">权重</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500">健康状态</th>
              <th className="text-center px-4 py-3 text-xs font-semibold text-gray-500">状态</th>
              <th className="text-right px-4 py-3 text-xs font-semibold text-gray-500">操作</th>
            </tr>
          </thead>
          <tbody>
            {providers.length === 0 ? (
              <tr><td colSpan={8} className="text-center py-12 text-gray-500">暂无 API 源</td></tr>
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
                    {p.enabled ? '已启用' : '已禁用'}
                  </button>
                </td>
                <td className="px-4 py-3 text-right">
                  <button onClick={() => openMappingModal(p)} title="模型映射" className="p-1 text-gray-400 hover:text-purple-600 ml-1"><GitBranch size={15} /></button>
                  <button onClick={() => handleEdit(p)} className="p-1 text-gray-400 hover:text-blue-600 ml-1"><Pencil size={15} /></button>
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
