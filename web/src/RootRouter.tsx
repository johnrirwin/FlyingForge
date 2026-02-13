import { useEffect, useRef } from 'react';
import { Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import App from './App';
import { AuthCallback } from './components';
import { LoginPage } from './components/LoginPage';
import { AUTH_EXPIRED_EVENT, type AuthExpiredEventDetail, buildLoginPath } from './authRouting';

export function RootRouter() {
  const navigate = useNavigate();
  const location = useLocation();
  const currentPathnameRef = useRef(location.pathname);

  useEffect(() => {
    currentPathnameRef.current = location.pathname;
  }, [location.pathname]);

  useEffect(() => {
    const handleAuthExpired = (event: Event) => {
      if (currentPathnameRef.current === '/login') {
        return;
      }

      const detail = (event as CustomEvent<AuthExpiredEventDetail>).detail;
      const nextPath = detail?.next ?? '/dashboard';
      navigate(buildLoginPath(nextPath, 'expired'), { replace: true });
    };

    window.addEventListener(AUTH_EXPIRED_EVENT, handleAuthExpired);
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleAuthExpired);
  }, [navigate]);

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/auth/callback" element={<AuthCallback />} />
      <Route path="/*" element={<App />} />
    </Routes>
  );
}
