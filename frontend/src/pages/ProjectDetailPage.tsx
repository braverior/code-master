import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { projectApi } from '@/api/project';
import { repositoryApi } from '@/api/repository';
import { requirementApi } from '@/api/requirement';
import { adminApi } from '@/api/admin';
import { feishuApi } from '@/api/feishu';
import { useAuth } from '@/hooks/use-auth';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Loading, Spinner } from '@/components/ui/spinner';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from '@/components/ui/dialog';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { DeleteConfirmDialog } from '@/components/DeleteConfirmDialog';
import {
  FolderKanban, GitBranch, Plus, Trash2, ExternalLink,
  RefreshCw, CheckCircle, XCircle, Clock, ArrowRight, Search, Archive,
  MoreHorizontal, Pencil, ChevronDown, ChevronRight, Code, FolderTree, Layers, UserPlus, Loader2, FileText, AlertTriangle,
} from 'lucide-react';
import type { Project, Repository, Requirement, User, AnalysisResult, DocLink } from '@/types';

export function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { user, isAdmin } = useAuth();
  const { toast } = useToast();
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('overview');

  const isOwner = project?.owner.id === user?.id;
  const canManage = isOwner || isAdmin;

  const fetchProject = useCallback(async () => {
    if (!id) return;
    try {
      const data = await projectApi.get(Number(id));
      setProject(data);
    } catch (err) {
      toast({ title: '获取项目详情失败', description: (err as Error).message, variant: 'destructive' });
    } finally {
      setLoading(false);
    }
  }, [id, toast]);

  useEffect(() => { fetchProject(); }, [fetchProject]);

  if (loading) return <Loading />;
  if (!project) return <div className="p-6 text-center text-muted-foreground">项目不存在</div>;

  const statusVariant = (status: string) => {
    switch (status) {
      case 'draft': return 'secondary' as const;
      case 'generating': return 'warning' as const;
      case 'generated': return 'default' as const;
      case 'reviewing': return 'outline' as const;
      case 'approved': case 'merged': return 'success' as const;
      case 'rejected': return 'destructive' as const;
      default: return 'secondary' as const;
    }
  };

  const statusLabel = (status: string) => {
    const map: Record<string, string> = {
      draft: '草稿', generating: '生成中', generated: '已生成',
      reviewing: '审查中', approved: '已通过', merged: '已合并', rejected: '已拒绝',
    };
    return map[status] || status;
  };

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <FolderKanban className="w-6 h-6 text-primary" />
            {project.name}
          </h1>
          <p className="text-muted-foreground mt-1">{project.description || '暂无描述'}</p>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={project.status === 'active' ? 'success' : 'secondary'}>
            {project.status === 'active' ? '活跃' : '已归档'}
          </Badge>
          {canManage && project.status === 'active' && (
            <Button variant="outline" size="sm" onClick={async () => {
              try {
                await projectApi.archive(project.id);
                toast({ title: '项目已归档', variant: 'success' });
                fetchProject();
              } catch (err) { toast({ title: '归档失败', description: (err as Error).message, variant: 'destructive' }); }
            }}>
              <Archive className="w-4 h-4 mr-1" />归档
            </Button>
          )}
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">概览</TabsTrigger>
          <TabsTrigger value="members">成员</TabsTrigger>
          <TabsTrigger value="repos">仓库</TabsTrigger>
          <TabsTrigger value="requirements">需求</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Card>
              <CardHeader><CardTitle className="text-lg">项目信息</CardTitle></CardHeader>
              <CardContent className="space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">创建者</span>
                  <span>{project.owner.name}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">成员数</span>
                  <span>{project.members?.length ?? 0}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">创建时间</span>
                  <span>{new Date(project.created_at).toLocaleDateString('zh-CN')}</span>
                </div>
                {project.doc_links && project.doc_links.length > 0 && (
                  <div>
                    <p className="text-sm text-muted-foreground mb-2">关联文档</p>
                    {project.doc_links.map((link, i) => (
                      <a key={i} href={link.url} target="_blank" rel="noreferrer"
                        className="flex items-center gap-1 text-sm text-primary hover:underline">
                        <ExternalLink className="w-3 h-3" />{link.title}
                      </a>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
            {project.stats && (
              <Card>
                <CardHeader><CardTitle className="text-lg">需求统计</CardTitle></CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 gap-3">
                    {Object.entries(project.stats).filter(([k]) => k !== 'total_requirements').map(([key, value]) => (
                      <div key={key} className="flex items-center justify-between p-2 rounded-md bg-muted/50">
                        <Badge variant={statusVariant(key)} className="text-xs">{statusLabel(key)}</Badge>
                        <span className="font-semibold">{value}</span>
                      </div>
                    ))}
                  </div>
                  <div className="mt-3 pt-3 border-t flex justify-between text-sm font-medium">
                    <span>总计</span>
                    <span>{project.stats.total_requirements}</span>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        {/* Members Tab */}
        <TabsContent value="members">
          <MembersTab project={project} canManage={canManage} onRefresh={fetchProject} />
        </TabsContent>

        {/* Repos Tab */}
        <TabsContent value="repos">
          <ReposTab projectId={project.id} />
        </TabsContent>

        {/* Requirements Tab */}
        <TabsContent value="requirements">
          <RequirementsTab project={project} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ---- Members Tab ----
function MembersTab({ project, canManage, onRefresh }: { project: Project; canManage: boolean; onRefresh: () => void }) {
  const { toast } = useToast();
  const [addOpen, setAddOpen] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchResults, setSearchResults] = useState<User[]>([]);
  const [searching, setSearching] = useState(false);
  const [addRole, setAddRole] = useState('rd');
  const [removeTarget, setRemoveTarget] = useState<number | null>(null);

  const handleSearch = async () => {
    if (!searchKeyword.trim()) return;
    setSearching(true);
    try {
      const results = await adminApi.searchUsers({ keyword: searchKeyword, exclude_project_id: project.id });
      setSearchResults(results);
    } catch { /* ignore */ }
    finally { setSearching(false); }
  };

  const handleAddMember = async (userId: number) => {
    try {
      await projectApi.addMembers(project.id, [userId], addRole);
      toast({ title: '成员添加成功', variant: 'success' });
      onRefresh();
      setSearchResults(searchResults.filter(u => u.id !== userId));
    } catch (err) { toast({ title: '添加失败', description: (err as Error).message, variant: 'destructive' }); }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-lg">项目成员</CardTitle>
        {canManage && (
          <Dialog open={addOpen} onOpenChange={setAddOpen}>
            <Button size="sm" onClick={() => setAddOpen(true)}><Plus className="w-4 h-4 mr-1" />添加成员</Button>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>添加成员</DialogTitle>
                <DialogDescription>搜索用户并添加到项目</DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="flex gap-2">
                  <Input placeholder="搜索姓名或邮箱" value={searchKeyword}
                    onChange={(e) => setSearchKeyword(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleSearch()} />
                  <Button onClick={handleSearch} disabled={searching}>
                    {searching ? <Spinner size="sm" /> : <Search className="w-4 h-4" />}
                  </Button>
                </div>
                <Select value={addRole} onValueChange={setAddRole}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="rd">开发工程师</SelectItem>
                    <SelectItem value="pm">产品经理</SelectItem>
                  </SelectContent>
                </Select>
                <div className="max-h-60 overflow-y-auto space-y-2">
                  {searchResults.map((u) => (
                    <div key={u.id} className="flex items-center justify-between p-2 rounded-md border">
                      <div className="flex items-center gap-2">
                        <div className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                          {u.name[0]}
                        </div>
                        <div>
                          <p className="text-sm font-medium">{u.name}</p>
                          <p className="text-xs text-muted-foreground">{u.email}</p>
                        </div>
                      </div>
                      <Button size="sm" onClick={() => handleAddMember(u.id)}>添加</Button>
                    </div>
                  ))}
                </div>
              </div>
            </DialogContent>
          </Dialog>
        )}
      </CardHeader>
      <CardContent>
        <table className="w-full">
          <thead>
            <tr className="border-b text-left text-sm text-muted-foreground">
              <th className="pb-2 font-medium">姓名</th>
              <th className="pb-2 font-medium">角色</th>
              <th className="pb-2 font-medium">加入时间</th>
              {canManage && <th className="pb-2 font-medium w-20">操作</th>}
            </tr>
          </thead>
          <tbody>
            {project.members?.map((member) => (
              <tr key={member.id} className="border-b last:border-0 hover:bg-muted/50">
                <td className="py-3">
                  <div className="flex items-center gap-2">
                    {member.avatar ? (
                      <img src={member.avatar} className="w-7 h-7 rounded-full" alt="" />
                    ) : (
                      <div className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                        {member.name[0]}
                      </div>
                    )}
                    <span className="text-sm">{member.name}</span>
                    {member.id === project.owner.id && <Badge variant="outline" className="text-[10px]">Owner</Badge>}
                  </div>
                </td>
                <td className="py-3"><Badge variant="secondary" className="text-xs">{member.role === 'pm' ? 'PM' : member.role === 'rd' ? 'RD' : member.role}</Badge></td>
                <td className="py-3 text-sm text-muted-foreground">{member.joined_at ? new Date(member.joined_at).toLocaleDateString('zh-CN') : '-'}</td>
                {canManage && (
                  <td className="py-3">
                    {member.id !== project.owner.id && (
                      <Button variant="ghost" size="icon" className="h-7 w-7 text-destructive"
                        onClick={() => setRemoveTarget(member.id)}>
                        <Trash2 className="w-3.5 h-3.5" />
                      </Button>
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </CardContent>
      <DeleteConfirmDialog
        open={removeTarget !== null}
        onOpenChange={() => setRemoveTarget(null)}
        title="移除成员"
        description="确定要将该成员从项目中移除吗？"
        onConfirm={async () => {
          if (removeTarget) {
            await projectApi.removeMember(project.id, removeTarget);
            onRefresh();
          }
        }}
      />
    </Card>
  );
}

// ---- Repos Tab ----
function ReposTab({ projectId }: { projectId: number }) {
  const { toast } = useToast();
  const [repos, setRepos] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [addOpen, setAddOpen] = useState(false);
  const [addLoading, setAddLoading] = useState(false);
  const [newRepo, setNewRepo] = useState({
    name: '', git_url: '', platform: 'gitlab', platform_project_id: '', default_branch: 'develop',
  });

  // Edit state
  const [editOpen, setEditOpen] = useState(false);
  const [editLoading, setEditLoading] = useState(false);
  const [editRepo, setEditRepo] = useState<{ id: number; name: string; default_branch: string } | null>(null);

  // Delete state
  const [deleteTarget, setDeleteTarget] = useState<Repository | null>(null);

  const fetchRepos = async () => {
    try {
      const data = await repositoryApi.listByProject(projectId);
      setRepos(data.list);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchRepos(); }, [projectId]);

  const handleAdd = async () => {
    setAddLoading(true);
    try {
      await repositoryApi.create(projectId, newRepo);
      toast({ title: '仓库关联成功', variant: 'success' });
      setAddOpen(false);
      setNewRepo({ name: '', git_url: '', platform: 'gitlab', platform_project_id: '', default_branch: 'develop' });
      fetchRepos();
    } catch (err) { toast({ title: '关联失败', description: (err as Error).message, variant: 'destructive' }); }
    finally { setAddLoading(false); }
  };

  const handleEdit = async () => {
    if (!editRepo) return;
    setEditLoading(true);
    try {
      const updates: Record<string, unknown> = { name: editRepo.name, default_branch: editRepo.default_branch };
      await repositoryApi.update(editRepo.id, updates);
      toast({ title: '仓库信息已更新', variant: 'success' });
      setEditOpen(false);
      setEditRepo(null);
      fetchRepos();
    } catch (err) { toast({ title: '更新失败', description: (err as Error).message, variant: 'destructive' }); }
    finally { setEditLoading(false); }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await repositoryApi.delete(deleteTarget.id);
      toast({ title: '仓库关联已解除', variant: 'success' });
      setDeleteTarget(null);
      fetchRepos();
    } catch (err) { toast({ title: '解除失败', description: (err as Error).message, variant: 'destructive' }); }
  };

  const handleAnalyze = async (repoId: number) => {
    try {
      await repositoryApi.triggerAnalysis(repoId);
      toast({ title: '分析任务已启动', variant: 'success' });
      fetchRepos();
    } catch (err) { toast({ title: '启动失败', description: (err as Error).message, variant: 'destructive' }); }
  };

  const analysisIcon = (status: string) => {
    switch (status) {
      case 'completed': return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'running': return <RefreshCw className="w-4 h-4 text-yellow-500 animate-spin" />;
      case 'failed': return <XCircle className="w-4 h-4 text-red-500" />;
      default: return <Clock className="w-4 h-4 text-muted-foreground" />;
    }
  };

  if (loading) return <Loading />;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-lg">代码仓库</CardTitle>
        <Dialog open={addOpen} onOpenChange={setAddOpen}>
          <Button size="sm" onClick={() => setAddOpen(true)}><Plus className="w-4 h-4 mr-1" />关联仓库</Button>
          <DialogContent className="max-w-xl">
            <DialogHeader>
              <DialogTitle>关联代码仓库</DialogTitle>
              <DialogDescription>添加项目的代码仓库</DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">仓库名称 *</label>
                  <Input placeholder="如: user-service" value={newRepo.name}
                    onChange={(e) => setNewRepo({ ...newRepo, name: e.target.value })} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">平台 *</label>
                  <Select value={newRepo.platform} onValueChange={(v) => setNewRepo({ ...newRepo, platform: v })}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="gitlab">GitLab</SelectItem>
                      <SelectItem value="github">GitHub</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Git URL *</label>
                <Input placeholder="https://gitlab.com/company/repo.git" value={newRepo.git_url}
                  onChange={(e) => setNewRepo({ ...newRepo, git_url: e.target.value })} />
              </div>
              {newRepo.platform === 'gitlab' && (
                <div className="space-y-2">
                  <label className="text-sm font-medium">GitLab Project ID *</label>
                  <Input placeholder="如: 12345" value={newRepo.platform_project_id}
                    onChange={(e) => setNewRepo({ ...newRepo, platform_project_id: e.target.value })} />
                </div>
              )}
              <div className="space-y-2">
                <label className="text-sm font-medium">默认分支</label>
                <Input placeholder="develop" value={newRepo.default_branch}
                  onChange={(e) => setNewRepo({ ...newRepo, default_branch: e.target.value })} />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setAddOpen(false)}>取消</Button>
              <Button onClick={handleAdd} disabled={addLoading || !newRepo.name || !newRepo.git_url}>
                {addLoading ? '关联中...' : '关联'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </CardHeader>
      <CardContent>
        {repos.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">暂无关联仓库</p>
        ) : (
          <div className="space-y-3">
            {repos.map((repo) => (
              <div key={repo.id} className="p-3 rounded-md border space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <GitBranch className="w-5 h-5 text-muted-foreground" />
                    <div>
                      <p className="text-sm font-medium">{repo.name}</p>
                      <p className="text-xs text-muted-foreground">{repo.git_url}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      {analysisIcon(repo.analysis_status)}
                      {repo.analysis_status === 'completed' ? '已分析' : repo.analysis_status === 'running' ? '分析中' : repo.analysis_status === 'failed' ? '分析失败' : '未分析'}
                    </div>
                    {repo.analysis_status !== 'running' && (
                      <Button variant="outline" size="sm" onClick={() => handleAnalyze(repo.id)}>
                        <RefreshCw className="w-3 h-3 mr-1" />分析
                      </Button>
                    )}
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="w-4 h-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => {
                          setEditRepo({ id: repo.id, name: repo.name, default_branch: repo.default_branch });
                          setEditOpen(true);
                        }}>
                          <Pencil className="w-4 h-4 mr-2" />修改
                        </DropdownMenuItem>
                        <DropdownMenuItem className="text-destructive" onClick={() => setDeleteTarget(repo)}>
                          <Trash2 className="w-4 h-4 mr-2" />解除关联
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
                {repo.analysis_status === 'failed' && repo.analysis_error && (
                  <div className="flex items-start gap-2 px-2 py-1.5 rounded bg-destructive/10 text-destructive text-xs">
                    <XCircle className="w-3.5 h-3.5 mt-0.5 shrink-0" />
                    <span className="break-all">{repo.analysis_error}</span>
                  </div>
                )}
                {repo.analysis_status === 'completed' && repo.analysis_result && (
                  <AnalysisResultPanel result={repo.analysis_result} />
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>

      {/* Edit Dialog */}
      <Dialog open={editOpen} onOpenChange={(open) => { setEditOpen(open); if (!open) setEditRepo(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>修改仓库信息</DialogTitle>
            <DialogDescription>修改仓库名称、默认分支</DialogDescription>
          </DialogHeader>
          {editRepo && (
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">仓库名称</label>
                <Input value={editRepo.name}
                  onChange={(e) => setEditRepo({ ...editRepo, name: e.target.value })} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">默认分支</label>
                <Input value={editRepo.default_branch}
                  onChange={(e) => setEditRepo({ ...editRepo, default_branch: e.target.value })} />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => { setEditOpen(false); setEditRepo(null); }}>取消</Button>
            <Button onClick={handleEdit} disabled={editLoading || !editRepo?.name}>
              {editLoading ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirm */}
      <DeleteConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={() => setDeleteTarget(null)}
        title="解除仓库关联"
        description={`确定要解除仓库「${deleteTarget?.name}」的关联吗？该仓库的分析结果也会一并删除。`}
        onConfirm={handleDelete}
      />
    </Card>
  );
}

// ---- Analysis Result Panel ----
function AnalysisResultPanel({ result }: { result: AnalysisResult }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border rounded-md overflow-hidden text-xs">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 p-2 bg-muted/30 hover:bg-muted/50 cursor-pointer"
      >
        {expanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
        <Code className="w-3.5 h-3.5 text-primary" />
        <span className="font-medium">分析结果</span>
        {result.tech_stack?.length > 0 && (
          <span className="text-muted-foreground ml-1">
            {result.tech_stack.slice(0, 4).join(' · ')}
            {result.tech_stack.length > 4 && ` +${result.tech_stack.length - 4}`}
          </span>
        )}
      </button>
      {expanded && (
        <div className="p-3 space-y-3 border-t">
          {/* Tech Stack */}
          {result.tech_stack?.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 text-muted-foreground font-semibold mb-1.5">
                <Layers className="w-3 h-3" />
                技术栈
              </div>
              <div className="flex flex-wrap gap-1">
                {result.tech_stack.map((tech) => (
                  <Badge key={tech} variant="secondary" className="text-[11px] px-1.5 py-0">
                    {tech}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {/* Modules */}
          {result.modules?.length > 0 && (
            <div>
              <div className="flex items-center gap-1.5 text-muted-foreground font-semibold mb-1.5">
                <FolderKanban className="w-3 h-3" />
                模块 ({result.modules.length})
              </div>
              <div className="space-y-1">
                {result.modules.map((m, i) => (
                  <div key={i} className="flex items-baseline gap-2">
                    <span className="font-mono text-primary shrink-0">{m.path}</span>
                    <span className="text-muted-foreground truncate">{m.description}</span>
                    <span className="text-muted-foreground/60 shrink-0 ml-auto">{m.files_count} 文件</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Directory Structure */}
          {result.directory_structure && (
            <div>
              <div className="flex items-center gap-1.5 text-muted-foreground font-semibold mb-1.5">
                <FolderTree className="w-3 h-3" />
                目录结构
              </div>
              <pre className="text-[11px] text-muted-foreground bg-muted/50 rounded p-2 whitespace-pre-wrap max-h-40 overflow-auto">
                {result.directory_structure}
              </pre>
            </div>
          )}

          {/* Entry Points */}
          {result.entry_points?.length > 0 && (
            <div>
              <div className="text-muted-foreground font-semibold mb-1">入口文件</div>
              <div className="flex flex-wrap gap-1">
                {result.entry_points.map((ep) => (
                  <span key={ep} className="font-mono text-primary bg-primary/10 rounded px-1.5 py-0.5">{ep}</span>
                ))}
              </div>
            </div>
          )}

          {/* Code Style */}
          {result.code_style && Object.values(result.code_style).some(v => v) && (
            <div>
              <div className="text-muted-foreground font-semibold mb-1">代码风格</div>
              <div className="space-y-0.5">
                {Object.entries(result.code_style).map(([key, value]) => (
                  value && (
                    <div key={key} className="flex gap-2">
                      <span className="text-muted-foreground/60 shrink-0">
                        {key === 'naming' ? '命名' : key === 'error_handling' ? '错误处理' : key === 'test_framework' ? '测试' : key}
                      </span>
                      <span className="text-foreground">{value}</span>
                    </div>
                  )
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ---- Requirements Tab ----
function RequirementsTab({ project }: { project: Project }) {
  const navigate = useNavigate();
  const { isPM, isAdmin } = useAuth();
  const { toast } = useToast();
  const [requirements, setRequirements] = useState<Requirement[]>([]);
  const [repos, setRepos] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState('all');
  const getDefaultDeadline = () => {
    const d = new Date();
    d.setDate(d.getDate() + 1);
    return d.toISOString().slice(0, 10);
  };

  const [createOpen, setCreateOpen] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [newReq, setNewReq] = useState({ title: '', description: '', priority: 'p1', repository_id: '', assignee_id: '', deadline: getDefaultDeadline() });
  const [newDocLinks, setNewDocLinks] = useState<DocLink[]>([]);
  const [newDocResolving, setNewDocResolving] = useState<Record<number, boolean>>({});
  const [newDocErrors, setNewDocErrors] = useState<Record<number, string>>({});

  const rdMembers = project.members || [];

  const fetchRequirements = async () => {
    try {
      const params: Record<string, unknown> = { page_size: 50 };
      if (statusFilter !== 'all') params.status = statusFilter;
      const data = await requirementApi.listByProject(project.id, params);
      setRequirements(data.list);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchRequirements(); }, [project.id, statusFilter]);

  useEffect(() => {
    repositoryApi.listByProject(project.id, { page_size: 100 })
      .then((data) => setRepos(data.list))
      .catch(() => {});
  }, [project.id]);

  const handleCreate = async () => {
    setCreateLoading(true);
    try {
      const data: Record<string, unknown> = {
        title: newReq.title,
        description: newReq.description,
        priority: newReq.priority,
      };
      if (newReq.repository_id) data.repository_id = Number(newReq.repository_id);
      if (newReq.assignee_id) data.assignee_id = Number(newReq.assignee_id);
      if (newReq.deadline) data.deadline = new Date(newReq.deadline + 'T23:59:59').toISOString();
      const validLinks = newDocLinks
        .filter((l) => l.url.trim())
        .map((l) => ({ ...l, title: l.title.trim() || l.url.trim() }));
      if (validLinks.length > 0) data.doc_links = validLinks;
      await requirementApi.create(project.id, data);
      toast({ title: '需求创建成功', variant: 'success' });
      setCreateOpen(false);
      setNewReq({ title: '', description: '', priority: 'p1', repository_id: '', assignee_id: '', deadline: getDefaultDeadline() });
      setNewDocLinks([]);
      fetchRequirements();
    } catch (err) { toast({ title: '创建失败', description: (err as Error).message, variant: 'destructive' }); }
    finally { setCreateLoading(false); }
  };

  const statusVariant = (status: string) => {
    switch (status) {
      case 'draft': return 'secondary' as const;
      case 'generating': return 'warning' as const;
      case 'generated': return 'default' as const;
      case 'reviewing': return 'outline' as const;
      case 'approved': case 'merged': return 'success' as const;
      case 'rejected': return 'destructive' as const;
      default: return 'secondary' as const;
    }
  };

  const statusLabel = (status: string) => {
    const map: Record<string, string> = {
      draft: '草稿', generating: '生成中', generated: '已生成',
      reviewing: '审查中', approved: '已通过', merged: '已合并', rejected: '已拒绝',
    };
    return map[status] || status;
  };

  const priorityVariant = (p: string) => {
    if (p === 'p0') return 'destructive' as const;
    if (p === 'p1') return 'warning' as const;
    return 'secondary' as const;
  };

  const handleQuickAssign = async (reqId: number, assigneeId: number) => {
    try {
      await requirementApi.update(reqId, { assignee_id: assigneeId });
      toast({ title: '指派成功', variant: 'success' });
      fetchRequirements();
    } catch (err) { toast({ title: '指派失败', description: (err as Error).message, variant: 'destructive' }); }
  };

  if (loading) return <Loading />;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div className="flex items-center gap-3">
          <CardTitle className="text-lg">需求列表</CardTitle>
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-28 h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部</SelectItem>
              <SelectItem value="draft">草稿</SelectItem>
              <SelectItem value="generating">生成中</SelectItem>
              <SelectItem value="generated">已生成</SelectItem>
              <SelectItem value="reviewing">审查中</SelectItem>
              <SelectItem value="approved">已通过</SelectItem>
              <SelectItem value="merged">已合并</SelectItem>
              <SelectItem value="rejected">已拒绝</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {(isPM || isAdmin) && project.status === 'active' && (
          <Dialog open={createOpen} onOpenChange={setCreateOpen}>
            <Button size="sm" onClick={() => setCreateOpen(true)}><Plus className="w-4 h-4 mr-1" />创建需求</Button>
            <DialogContent className="max-w-xl max-h-[85vh] flex flex-col">
              <DialogHeader>
                <DialogTitle>创建需求</DialogTitle>
                <DialogDescription>创建新的需求</DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4 overflow-y-auto flex-1 min-h-0">
                <div className="space-y-2">
                  <label className="text-sm font-medium">标题 *</label>
                  <Input placeholder="需求标题" value={newReq.title}
                    onChange={(e) => setNewReq({ ...newReq, title: e.target.value })} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">描述 *</label>
                  <Textarea placeholder="详细描述需求（支持 Markdown）" value={newReq.description}
                    onChange={(e) => setNewReq({ ...newReq, description: e.target.value })} rows={6} />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">优先级</label>
                    <Select value={newReq.priority} onValueChange={(v) => setNewReq({ ...newReq, priority: v })}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="p0">P0 - 紧急</SelectItem>
                        <SelectItem value="p1">P1 - 高</SelectItem>
                        <SelectItem value="p2">P2 - 中</SelectItem>
                        <SelectItem value="p3">P3 - 低</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">目标仓库</label>
                    <Select value={newReq.repository_id} onValueChange={(v) => setNewReq({ ...newReq, repository_id: v })}>
                      <SelectTrigger><SelectValue placeholder="选择仓库" /></SelectTrigger>
                      <SelectContent>
                        {repos.map((r) => (
                          <SelectItem key={r.id} value={String(r.id)}>{r.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">指派研发</label>
                  <Select value={newReq.assignee_id} onValueChange={(v) => setNewReq({ ...newReq, assignee_id: v })}>
                    <SelectTrigger><SelectValue placeholder="选择研发人员" /></SelectTrigger>
                    <SelectContent>
                      {rdMembers.map((m) => (
                        <SelectItem key={m.id} value={String(m.id)}>{m.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">可选，指派后可直接触发代码生成</p>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">期望完成时间</label>
                  <Input type="date" value={newReq.deadline}
                    onClick={(e) => (e.target as HTMLInputElement).showPicker?.()}
                    onChange={(e) => setNewReq({ ...newReq, deadline: e.target.value })} />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium">关联文档</label>
                  {newDocLinks.map((link, i) => (
                    <div key={i} className="flex items-center gap-2">
                      <div className="flex-1 space-y-1">
                        <Input placeholder="粘贴飞书文档链接" value={link.url}
                          onChange={(e) => {
                            const updated = [...newDocLinks];
                            updated[i] = { ...updated[i], url: e.target.value };
                            setNewDocLinks(updated);
                          }}
                          onBlur={async () => {
                            const url = link.url.trim();
                            if (!url || link.title) return;
                            setNewDocResolving((prev) => ({ ...prev, [i]: true }));
                            setNewDocErrors((prev) => { const next = { ...prev }; delete next[i]; return next; });
                            try {
                              const res = await feishuApi.resolveDoc(url);
                              setNewDocLinks((prev) => {
                                const updated = [...prev];
                                if (updated[i]) updated[i] = { ...updated[i], title: res.title };
                                return updated;
                              });
                            } catch (err) {
                              const msg = (err as Error).message || '';
                              if (msg.includes('forBidden') || msg.includes('forbidden') || msg.includes('权限')) {
                                setNewDocErrors((prev) => ({ ...prev, [i]: '无权访问该文档，请在飞书中将文档设为「组织内链接可阅读」' }));
                              } else {
                                setNewDocErrors((prev) => ({ ...prev, [i]: '文档解析失败，保存后将使用链接作为标题' }));
                              }
                            }
                            finally { setNewDocResolving((prev) => ({ ...prev, [i]: false })); }
                          }} />
                        {newDocResolving[i] ? (
                          <div className="flex items-center gap-1 text-xs text-muted-foreground">
                            <Loader2 className="w-3 h-3 animate-spin" />解析中...
                          </div>
                        ) : newDocErrors[i] ? (
                          <p className="text-xs text-destructive">{newDocErrors[i]}</p>
                        ) : link.title ? (
                          <p className="text-xs text-muted-foreground truncate">{link.title}</p>
                        ) : null}
                      </div>
                      <Button variant="ghost" size="icon" className="shrink-0 h-9 w-9 text-muted-foreground hover:text-destructive"
                        onClick={() => {
                          setNewDocLinks(newDocLinks.filter((_, idx) => idx !== i));
                          setNewDocResolving((prev) => { const next = { ...prev }; delete next[i]; return next; });
                          setNewDocErrors((prev) => { const next = { ...prev }; delete next[i]; return next; });
                        }}>
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  ))}
                  <Button variant="outline" size="sm" className="w-full"
                    onClick={() => setNewDocLinks([...newDocLinks, { title: '', url: '' }])}>
                    <Plus className="w-4 h-4 mr-1" />添加文档
                  </Button>
                  <p className="text-xs text-muted-foreground">粘贴飞书文档链接后自动解析标题</p>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setCreateOpen(false)}>取消</Button>
                <Button onClick={handleCreate} disabled={createLoading || !newReq.title.trim() || !newReq.description.trim()}>
                  {createLoading ? '创建中...' : '创建'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        )}
      </CardHeader>
      <CardContent>
        {requirements.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">暂无需求</p>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b text-left text-sm text-muted-foreground">
                <th className="pb-2 font-medium">标题</th>
                <th className="pb-2 font-medium w-20">优先级</th>
                <th className="pb-2 font-medium w-24">状态</th>
                <th className="pb-2 font-medium w-28">截止时间</th>
                <th className="pb-2 font-medium w-16">指派人</th>
                <th className="pb-2 font-medium w-24">仓库</th>
                <th className="pb-2 font-medium w-10"></th>
              </tr>
            </thead>
            <tbody>
              {requirements.map((req) => (
                <tr key={req.id} className="border-b last:border-0 hover:bg-muted/50 cursor-pointer"
                  onClick={() => navigate(`/requirements/${req.id}`)}>
                  <td className="py-3 text-sm">
                    <span className="inline-flex items-center gap-1.5">
                      <span className="text-muted-foreground shrink-0">#{req.id}</span>
                      {req.title}
                      {req.doc_links && req.doc_links.length > 0 && (
                        <span title="已关联文档"><FileText className="w-3.5 h-3.5 text-primary/60 shrink-0" /></span>
                      )}
                    </span>
                  </td>
                  <td className="py-3"><Badge variant={priorityVariant(req.priority)} className="text-xs">{req.priority.toUpperCase()}</Badge></td>
                  <td className="py-3"><Badge variant={statusVariant(req.status)} className="text-xs">{statusLabel(req.status)}</Badge></td>
                  <td className="py-3 text-sm text-muted-foreground">
                    {req.deadline ? (
                      <span className={`inline-flex items-center gap-1 ${
                        new Date(req.deadline) < new Date() && req.status !== 'merged' && req.status !== 'approved'
                          ? 'text-destructive font-medium' : ''
                      }`}>
                        {new Date(req.deadline).toLocaleDateString('zh-CN')}
                        {new Date(req.deadline) < new Date() && req.status !== 'merged' && req.status !== 'approved' && (
                          <AlertTriangle className="w-3.5 h-3.5" />
                        )}
                      </span>
                    ) : '-'}
                  </td>
                  <td className="py-3">
                    {req.assignee ? (
                      req.assignee.avatar ? (
                        <img src={req.assignee.avatar} alt={req.assignee.name} title={req.assignee.name} className="w-7 h-7 rounded-full" />
                      ) : (
                        <div title={req.assignee.name} className="w-7 h-7 rounded-full bg-primary/20 flex items-center justify-center text-xs font-medium">
                          {req.assignee.name[0]}
                        </div>
                      )
                    ) : <span className="text-sm text-muted-foreground">-</span>}
                  </td>
                  <td className="py-3 text-sm text-muted-foreground">{req.repository?.name || '-'}</td>
                  <td className="py-3">
                    {!req.assignee && (req.status === 'draft' || req.status === 'rejected') && rdMembers.length > 0 ? (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm" className="h-7 px-2 text-xs text-primary"
                            onClick={(e) => e.stopPropagation()}>
                            <UserPlus className="w-3.5 h-3.5 mr-1" />指派
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
                          {rdMembers.map((m) => (
                            <DropdownMenuItem key={m.id} onClick={() => handleQuickAssign(req.id, m.id)}>
                              <div className="flex items-center gap-2">
                                <div className="w-5 h-5 rounded-full bg-primary/20 flex items-center justify-center text-[10px] font-medium">
                                  {m.name[0]}
                                </div>
                                {m.name}
                              </div>
                            </DropdownMenuItem>
                          ))}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    ) : (
                      <ArrowRight className="w-4 h-4 text-muted-foreground" />
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  );
}
