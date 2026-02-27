import { useState, useEffect, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '@/hooks/use-auth';
import { authApi } from '@/api/auth';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Code2 } from 'lucide-react';

export function LoginPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { login, isAuthenticated } = useAuth();

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showRoleDialog, setShowRoleDialog] = useState(false);
  const [roleLoading, setRoleLoading] = useState(false);

  const loginAttempted = useRef(false);

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/', { replace: true });
    }
  }, [isAuthenticated, navigate]);

  // Handle OAuth callback - token comes in URL
  useEffect(() => {
    const token = searchParams.get('token');
    const isNewUser = searchParams.get('is_new_user');

    if (token && !loginAttempted.current) {
      loginAttempted.current = true;
      setSearchParams({}, { replace: true });

      setLoading(true);
      // Fetch user info with the token
      localStorage.setItem('token', token);
      authApi.getMe()
        .then((user) => {
          login(token, user);
          if (isNewUser === 'true') {
            setShowRoleDialog(true);
          } else {
            navigate('/', { replace: true });
          }
        })
        .catch((err) => {
          setError(err instanceof Error ? err.message : '登录失败');
          localStorage.removeItem('token');
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [searchParams, setSearchParams, login, navigate]);

  const handleFeishuLogin = () => {
    // Redirect to backend's feishu login endpoint
    const redirectUri = encodeURIComponent(window.location.origin + '/login');
    window.location.href = `/api/v1/auth/feishu/login?redirect_uri=${redirectUri}`;
  };

  const handleSelectRole = async (role: string) => {
    setRoleLoading(true);
    try {
      const updatedUser = await authApi.selectRole(role);
      const currentUser = JSON.parse(localStorage.getItem('user') || '{}');
      const merged = { ...currentUser, ...updatedUser, is_new_user: false };
      localStorage.setItem('user', JSON.stringify(merged));
      setShowRoleDialog(false);
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '角色选择失败');
    } finally {
      setRoleLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="flex justify-center mb-4">
            <div className="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center">
              <Code2 className="w-10 h-10 text-primary" />
            </div>
          </div>
          <CardTitle className="text-2xl">CodeMaster</CardTitle>
          <CardDescription>
            AI 驱动的代码生成平台
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {loading ? (
            <div className="flex flex-col items-center py-8">
              <Spinner size="lg" />
              <p className="mt-4 text-muted-foreground">登录中...</p>
            </div>
          ) : (
            <>
              {error && (
                <div className="p-3 rounded-md bg-destructive/10 text-destructive text-sm">
                  {error}
                </div>
              )}
              <Button
                className="w-full h-12"
                onClick={handleFeishuLogin}
              >
                <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M3 3l8.735 4.56L21 3v8.462L11.735 16 3 11.462V3zm0 10.154L11.735 17.7 21 13.154V21l-9.265-4.538L3 21v-7.846z" />
                </svg>
                飞书登录
              </Button>
              <p className="text-center text-xs text-muted-foreground">
                使用公司飞书账号登录
              </p>
            </>
          )}
        </CardContent>
      </Card>

      {/* Role Selection Dialog */}
      <Dialog open={showRoleDialog} onOpenChange={setShowRoleDialog}>
        <DialogContent hideCloseButton>
          <DialogHeader>
            <DialogTitle>选择你的角色</DialogTitle>
            <DialogDescription>
              首次登录，请选择你的角色。角色确定后如需修改请联系管理员。
            </DialogDescription>
          </DialogHeader>
          <div className="grid grid-cols-2 gap-4 py-4">
            <Button
              variant="outline"
              className="h-24 flex flex-col gap-2"
              onClick={() => handleSelectRole('pm')}
              disabled={roleLoading}
            >
              <span className="text-2xl">📋</span>
              <span className="font-medium">产品经理 (PM)</span>
            </Button>
            <Button
              variant="outline"
              className="h-24 flex flex-col gap-2"
              onClick={() => handleSelectRole('rd')}
              disabled={roleLoading}
            >
              <span className="text-2xl">💻</span>
              <span className="font-medium">开发工程师 (RD)</span>
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
