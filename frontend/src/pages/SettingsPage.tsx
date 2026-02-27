import { useState, useEffect, useRef } from 'react';
import { settingApi, type LLMSettings } from '@/api/setting';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import { Loading } from '@/components/ui/spinner';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Settings, Eye, EyeOff, Save, ChevronsUpDown, Check } from 'lucide-react';

const DEFAULT_BASE_URL = 'https://bmc-llm-relay.bluemediagroup.cn/v1';

const PRESET_MODELS = [
  { value: 'claude-sonnet-4-5-20250929', label: 'Claude Sonnet 4.5' },
  { value: 'claude-haiku-4-5-20251001', label: 'Claude Haiku 4.5' },
  { value: 'claude-opus-4-5-20251101', label: 'Claude Opus 4.5' },
  { value: 'claude-opus-4-6-v1', label: 'Claude Opus 4.6' },
];

export function SettingsPage() {
  const { toast } = useToast();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showKey, setShowKey] = useState(false);
  const [showGitlabToken, setShowGitlabToken] = useState(false);
  const [form, setForm] = useState<LLMSettings>({
    base_url: '',
    api_key: '',
    model: '',
    gitlab_token: '',
  });

  useEffect(() => {
    (async () => {
      try {
        const data = await settingApi.getLLM();
        setForm({
          base_url: data.base_url || '',
          api_key: data.api_key || '',
          model: data.model || '',
          gitlab_token: data.gitlab_token || '',
        });
      } catch (err) {
        toast({ title: '获取设置失败', description: (err as Error).message, variant: 'destructive' });
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const handleSave = async () => {
    setSaving(true);
    try {
      const data = await settingApi.updateLLM(form);
      setForm({
        base_url: data.base_url || '',
        api_key: data.api_key || '',
        model: data.model || '',
        gitlab_token: data.gitlab_token || '',
      });
      toast({ title: '保存成功', description: '配置已更新' });
    } catch (err) {
      toast({ title: '保存失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Loading />;

  return (
    <div className="p-6 max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Settings className="w-6 h-6" />
          设置
        </h1>
        <p className="text-muted-foreground mt-1">管理你的 Claude Code 和 Git 配置</p>
      </div>

      {/* LLM 配置 */}
      <Card>
        <CardContent className="p-6 space-y-6">
          <div>
            <h2 className="text-lg font-semibold">Claude Code 接口配置</h2>
            <p className="text-sm text-muted-foreground mt-1">
              以下配置将作为环境变量注入 Claude Code 进程，用于代码生成和 AI Review。未填写时将使用系统默认配置。
            </p>
          </div>

          <Separator />

          {/* Base URL */}
          <div className="space-y-2">
            <label className="text-sm font-medium">API Base URL</label>
            <Input
              placeholder={DEFAULT_BASE_URL}
              value={form.base_url}
              onChange={(e) => setForm({ ...form, base_url: e.target.value })}
            />
            <p className="text-xs text-muted-foreground">
              Anthropic 兼容的 API 地址，将注入为 ANTHROPIC_BASE_URL。默认: {DEFAULT_BASE_URL}
            </p>
          </div>

          {/* API Key */}
          <div className="space-y-2">
            <label className="text-sm font-medium">API Key</label>
            <div className="relative">
              <Input
                type={showKey ? 'text' : 'password'}
                placeholder="sk-..."
                value={form.api_key}
                onChange={(e) => setForm({ ...form, api_key: e.target.value })}
                className="pr-10"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7"
                onClick={() => setShowKey(!showKey)}
              >
                {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              将注入为 ANTHROPIC_API_KEY，用于 Claude Code 调用大模型时鉴权
            </p>
          </div>

          {/* Model */}
          <ModelCombobox
            value={form.model}
            onChange={(v) => setForm({ ...form, model: v })}
          />
        </CardContent>
      </Card>

      {/* Git Token 配置 */}
      <Card>
        <CardContent className="p-6 space-y-6">
          <div>
            <h2 className="text-lg font-semibold">Git Token 配置</h2>
            <p className="text-sm text-muted-foreground mt-1">
              配置 Git Personal Access Token，用于代码仓库的克隆、推送等操作
            </p>
          </div>

          <Separator />

          {/* Git Token */}
          <div className="space-y-2">
            <label className="text-sm font-medium">Git Token</label>
            <div className="relative">
              <Input
                type={showGitlabToken ? 'text' : 'password'}
                placeholder="glpat-... 或 ghp_..."
                value={form.gitlab_token}
                onChange={(e) => setForm({ ...form, gitlab_token: e.target.value })}
                className="pr-10"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-1 top-1/2 -translate-y-1/2 h-7 w-7"
                onClick={() => setShowGitlabToken(!showGitlabToken)}
              >
                {showGitlabToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">
              GitLab / GitHub Personal Access Token，需要 read_repository、write_repository 权限
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Save Button */}
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={saving}>
          <Save className="w-4 h-4 mr-2" />
          {saving ? '保存中...' : '保存配置'}
        </Button>
      </div>
    </div>
  );
}

function ModelCombobox({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [open, setOpen] = useState(false);
  const [inputValue, setInputValue] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  const isPreset = PRESET_MODELS.some((m) => m.value === value);
  const displayLabel = PRESET_MODELS.find((m) => m.value === value)?.label;

  const filtered = PRESET_MODELS.filter(
    (m) => !inputValue || m.value.includes(inputValue.toLowerCase()) || m.label.toLowerCase().includes(inputValue.toLowerCase()),
  );

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">模型名称</label>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            className="w-full justify-between font-normal h-10"
            onClick={() => {
              setInputValue('');
              setOpen(true);
            }}
          >
            <span className={value ? 'text-foreground' : 'text-muted-foreground'}>
              {value ? (isPreset ? `${displayLabel} (${value})` : value) : '选择或输入模型名称...'}
            </span>
            <ChevronsUpDown className="w-4 h-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start">
          <div className="p-2 border-b border-border">
            <Input
              ref={inputRef}
              placeholder="搜索或输入自定义模型..."
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && inputValue.trim()) {
                  onChange(inputValue.trim());
                  setOpen(false);
                }
              }}
              className="h-8"
              autoFocus
            />
          </div>
          <div className="max-h-52 overflow-y-auto p-1">
            {filtered.map((m) => (
              <button
                key={m.value}
                className="flex items-center gap-2 w-full px-2 py-1.5 text-sm rounded-sm hover:bg-muted cursor-pointer text-left"
                onClick={() => {
                  onChange(m.value);
                  setOpen(false);
                }}
              >
                <Check className={`w-4 h-4 shrink-0 ${value === m.value ? 'opacity-100' : 'opacity-0'}`} />
                <span className="flex-1">{m.label}</span>
                <span className="text-xs text-muted-foreground">{m.value}</span>
              </button>
            ))}
            {inputValue.trim() && !PRESET_MODELS.some((m) => m.value === inputValue.trim()) && (
              <button
                className="flex items-center gap-2 w-full px-2 py-1.5 text-sm rounded-sm hover:bg-muted cursor-pointer text-left"
                onClick={() => {
                  onChange(inputValue.trim());
                  setOpen(false);
                }}
              >
                <Check className={`w-4 h-4 shrink-0 ${value === inputValue.trim() ? 'opacity-100' : 'opacity-0'}`} />
                <span className="flex-1">使用自定义模型</span>
                <span className="text-xs text-muted-foreground">{inputValue.trim()}</span>
              </button>
            )}
            {filtered.length === 0 && !inputValue.trim() && (
              <p className="px-2 py-4 text-sm text-center text-muted-foreground">无匹配模型</p>
            )}
          </div>
        </PopoverContent>
      </Popover>
      <p className="text-xs text-muted-foreground">
        留空则使用系统默认模型。该值不会直接注入环境变量，仅用于平台内 AI 对话功能
      </p>
    </div>
  );
}
