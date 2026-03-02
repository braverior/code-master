import { createContext, useContext, useState, useEffect, useCallback, useRef, type ReactNode } from 'react';
import { authApi } from '@/api/auth';
import type { User } from '@/types';

interface AuthContextType {
  user: User | null;
  token: string | null;
  loading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isPM: boolean;
  isRD: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
  setUser: (user: User) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUserState] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const refreshingRef = useRef(false);

  useEffect(() => {
    const storedToken = localStorage.getItem('token');
    const storedUser = localStorage.getItem('user');

    if (storedToken && storedUser) {
      setToken(storedToken);
      try {
        setUserState(JSON.parse(storedUser));
      } catch {
        localStorage.removeItem('user');
      }
    }
    setLoading(false);
  }, []);

  const login = useCallback((newToken: string, newUser: User) => {
    localStorage.setItem('token', newToken);
    localStorage.setItem('user', JSON.stringify(newUser));
    setToken(newToken);
    setUserState(newUser);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    setToken(null);
    setUserState(null);
  }, []);

  const setUser = useCallback((u: User) => {
    setUserState(u);
    localStorage.setItem('user', JSON.stringify(u));
  }, []);

  const refreshUser = useCallback(async () => {
    if (!token || refreshingRef.current) return;
    refreshingRef.current = true;
    try {
      const userData = await authApi.getMe();
      setUserState(userData);
      localStorage.setItem('user', JSON.stringify(userData));
    } catch {
      logout();
    } finally {
      refreshingRef.current = false;
    }
  }, [token, logout]);

  // Auto-refresh user info when page becomes visible (tab switch back, window focus)
  useEffect(() => {
    if (!token) return;

    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        refreshUser();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [token, refreshUser]);

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        loading,
        isAuthenticated: !!token && !!user,
        isAdmin: user?.is_admin === true,
        isPM: user?.role === 'pm',
        isRD: user?.role === 'rd',
        login,
        logout,
        refreshUser,
        setUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
