import { Link, useLocation } from 'react-router-dom';
import { cn } from '@/lib/utils';
import { Separator } from '@/components/ui/separator';
import { Button } from '@/components/ui/button';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/hooks/use-auth';
import { useTheme } from '@/hooks/use-theme';
import {
  Code2,
  LayoutDashboard,
  FolderKanban,
  ClipboardCheck,
  Users,
  FileText,
  LogOut,
  ShieldCheck,
  Mail,
  Sun,
  Moon,
  Settings,
} from 'lucide-react';

const navItems = [
  { path: '/', label: '仪表盘', icon: LayoutDashboard },
  { path: '/projects', label: '项目', icon: FolderKanban },
  { path: '/requirements', label: '需求', icon: FileText },
  { path: '/reviews', label: '审查列表', icon: ClipboardCheck },
  { path: '/settings', label: '设置', icon: Settings },
];

const adminNavItems = [
  { path: '/admin/users', label: '用户管理', icon: Users },
];

export function Sidebar() {
  const location = useLocation();
  const { user, isAdmin, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();

  const handleLogout = () => {
    logout();
    window.location.href = '/login';
  };

  const roleLabel = (role?: string) => {
    switch (role) {
      case 'pm': return '产品经理';
      case 'rd': return '开发工程师';
      default: return role;
    }
  };

  return (
    <aside className="w-64 h-screen sticky top-0 bg-card border-r border-border flex flex-col">
      {/* Logo */}
      <div className="p-4 flex items-center gap-3">
        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
          <Code2 className="w-6 h-6 text-primary" />
        </div>
        <div>
          <h1 className="font-semibold text-foreground">CodeMaster</h1>
          <p className="text-xs text-muted-foreground">AI 代码生成平台</p>
        </div>
      </div>

      <Separator />

      {/* Navigation */}
      <nav className="flex-1 p-3 overflow-y-auto">
        <ul className="space-y-1">
          {navItems.map((item) => {
            const isActive = item.path === '/'
              ? location.pathname === '/'
              : location.pathname.startsWith(item.path);
            const Icon = item.icon;
            return (
              <li key={item.path}>
                <Link
                  to={item.path}
                  className={cn(
                    'flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors cursor-pointer',
                    isActive
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  )}
                >
                  <Icon className="w-4 h-4" />
                  {item.label}
                </Link>
              </li>
            );
          })}
        </ul>

        {/* Admin Section */}
        {isAdmin && (
          <>
            <Separator className="my-3" />
            <p className="px-3 mb-2 text-xs font-medium text-muted-foreground flex items-center gap-1.5">
              <ShieldCheck className="w-3.5 h-3.5" />
              管理
            </p>
            <ul className="space-y-1">
              {adminNavItems.map((item) => {
                const isActive = location.pathname.startsWith(item.path);
                const Icon = item.icon;
                return (
                  <li key={item.path}>
                    <Link
                      to={item.path}
                      className={cn(
                        'flex items-center gap-3 px-3 py-2.5 rounded-md text-sm font-medium transition-colors cursor-pointer',
                        isActive
                          ? 'bg-primary/10 text-primary'
                          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                      )}
                    >
                      <Icon className="w-4 h-4" />
                      {item.label}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </>
        )}
      </nav>

      {/* User Profile & Footer */}
      <div className="p-3 border-t border-border">
        {user && (
          <Popover>
            <div className="flex items-center gap-2">
              <PopoverTrigger asChild>
                <div className="flex items-center gap-3 flex-1 min-w-0 cursor-pointer hover:bg-muted/50 rounded-md p-1.5 -m-1.5 transition-colors">
                  {user.avatar ? (
                    <img
                      src={user.avatar}
                      alt={user.name}
                      className="w-8 h-8 rounded-full ring-2 ring-border"
                    />
                  ) : (
                    <div className="w-8 h-8 rounded-full bg-gradient-to-br from-primary/80 to-primary flex items-center justify-center ring-2 ring-border">
                      <span className="text-xs font-medium text-primary-foreground">{user.name[0]}</span>
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{user.name}</p>
                    <span className="text-[10px] text-muted-foreground">
                      {roleLabel(user.role)}{user.is_admin ? ' · 管理员' : ''}
                    </span>
                  </div>
                </div>
              </PopoverTrigger>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0 text-muted-foreground hover:text-foreground"
                onClick={toggleTheme}
                title={theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'}
              >
                {theme === 'dark' ? <Sun className="w-3.5 h-3.5" /> : <Moon className="w-3.5 h-3.5" />}
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive"
                onClick={handleLogout}
                title="退出登录"
              >
                <LogOut className="w-3.5 h-3.5" />
              </Button>
            </div>
            <PopoverContent className="w-72 p-0" side="top" align="start">
              <div className="p-4 bg-gradient-to-br from-primary/10 to-primary/5">
                <div className="flex items-center gap-3">
                  {user.avatar ? (
                    <img
                      src={user.avatar}
                      alt={user.name}
                      className="w-12 h-12 rounded-full ring-2 ring-background shadow-md"
                    />
                  ) : (
                    <div className="w-12 h-12 rounded-full bg-gradient-to-br from-primary to-primary/80 flex items-center justify-center ring-2 ring-background shadow-md">
                      <span className="text-lg font-semibold text-primary-foreground">{user.name[0]}</span>
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="font-semibold truncate">{user.name}</p>
                    <div className="flex items-center gap-1 mt-1">
                      <Badge variant="secondary" className="text-[10px]">
                        {roleLabel(user.role)}
                      </Badge>
                      {user.is_admin && (
                        <Badge variant="default" className="text-[10px]">
                          <ShieldCheck className="w-3 h-3 mr-1" />管理员
                        </Badge>
                      )}
                    </div>
                  </div>
                </div>
              </div>
              <div className="p-3 space-y-2.5">
                {user.email && (
                  <div className="flex items-center gap-2.5 text-sm">
                    <Mail className="w-4 h-4 text-muted-foreground shrink-0" />
                    <span className="text-muted-foreground truncate">{user.email}</span>
                  </div>
                )}
              </div>
              <Separator />
              <div className="p-2">
                <Button
                  variant="ghost"
                  className="w-full justify-start text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={handleLogout}
                >
                  <LogOut className="w-4 h-4 mr-2" />
                  退出登录
                </Button>
              </div>
            </PopoverContent>
          </Popover>
        )}
      </div>
    </aside>
  );
}
