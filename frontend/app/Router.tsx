import React, { useEffect, useRef } from 'react';
import { RouteComponentProps, withRouter } from 'react-router-dom';

import IFrameRoutes from 'App/IFrameRoutes';
import PrivateRoutes from 'App/PrivateRoutes';
import PublicRoutes from 'App/PublicRoutes';
import {
  GLOBAL_DESTINATION_PATH,
  IFRAME,
  SITE_ID_STORAGE_KEY,
} from 'App/constants/storageKeys';
import Layout from 'App/layout/Layout';
import { useStore } from 'App/mstore';
import { checkParam } from 'App/utils';
import { ModalProvider } from 'Components/Modal';
import { ModalProvider as NewModalProvider } from 'Components/ModalContext';
import { Loader } from 'UI';
import { observer } from 'mobx-react-lite';
import * as routes from './routes';
import Tracker from 'App/Tracker';

interface RouterProps extends RouteComponentProps<{ siteId: string }> {}

const Router: React.FC<RouterProps> = (props) => {
  const { location, history } = props;
  const mstore = useStore();
  const {
    customFieldStore,
    projectsStore,
    sessionStore,
    searchStore,
    userStore,
    settingsStore,
  } = mstore;
  const { jwt } = userStore;
  const { changePassword } = userStore.account;
  const userInfoLoading = userStore.fetchInfoRequest.loading;
  const isLoggedIn = Boolean(jwt && !changePassword);
  const { fetchUserInfo } = userStore;
  const setJwt = userStore.updateJwt;
  const { logout } = userStore;

  const { setSessionPath } = sessionStore;
  const { siteId } = projectsStore;
  const { sitesLoading } = projectsStore;
  const sites = projectsStore.list;
  const loading = Boolean(userInfoLoading || sitesLoading);
  const initSite = projectsStore.initProject;
  const fetchSiteList = projectsStore.fetchList;

  const [isSignup, setIsSignup] = React.useState(false);
  const [isIframe, setIsIframe] = React.useState(false);


  const handleDestinationPath = () => {
    if (!isLoggedIn && location.pathname !== routes.login()) {
      localStorage.setItem(
        GLOBAL_DESTINATION_PATH,
        location.pathname + location.search,
      );
    }
  };

  const handleUserLogin = async () => {
    const userData = await fetchUserInfo();
    const siteIdFromPath = location.pathname.split('/')[1];
    await fetchSiteList(siteIdFromPath, userData?.tenantId);
    mstore.initClient();

    if (userData?.tenantId) {
      projectsStore.setTenantId(userData.tenantId);
      const existing = localStorage.getItem(SITE_ID_STORAGE_KEY);
      const [storedSiteId, storedTenantId] = existing
        ? existing.split('_$_')
        : [null, null];

      if (userData?.tenantId === storedTenantId) {
        projectsStore.setSiteId(storedSiteId!);
      } else {
        localStorage.setItem(
          SITE_ID_STORAGE_KEY,
          `${projectsStore.siteId}_$_${userData.tenantId}`,
        );
      }
    }

    // Replay-only build: Spot disabled (kept for later).
    // if (localSpotJwt && !isTokenExpired(localSpotJwt)) {
    //   // handleSpotLogin(localSpotJwt);
    // }

    const destinationPath = localStorage.getItem(GLOBAL_DESTINATION_PATH);
    if (
      destinationPath &&
      !destinationPath.includes(routes.login()) &&
      !destinationPath.includes(routes.signup()) &&
      destinationPath !== '/'
    ) {
      const url = new URL(destinationPath, window.location.origin);
      checkParams(url.search);
      history.push(destinationPath);
      localStorage.removeItem(GLOBAL_DESTINATION_PATH);
    }
  };

  const checkParams = (search?: string) => {
    const _isIframe = checkParam('iframe', IFRAME, search);
    setIsIframe(_isIframe);
  };

  useEffect(() => {
    checkParams();
    mstore.initClient();

    const vmodeParam = new URLSearchParams(location.search).get('vmode');
    if (vmodeParam) {
      settingsStore.sessionSettings.updateKey('virtualMode', true);
    }
  }, []);

  useEffect(() => {
    if (location.pathname.includes('signup')) {
      setIsSignup(true);
    }
  }, [location.pathname]);

  useEffect(() => {
    handleDestinationPath();

    setSessionPath(previousLocation || location);
  }, [location]);

  useEffect(() => {
    if (prevIsLoggedIn !== isLoggedIn && isLoggedIn) {
      void handleUserLogin();
    }
  }, [isLoggedIn]);

  useEffect(() => {
    // Replay-only build: Spot disabled (kept for later).
    // if (isLoggedIn && isSpotCb && !isSignup) {
    //   if (localSpotJwt && !isTokenExpired(localSpotJwt)) {
    //     // handleSpotLogin(localSpotJwt);
    //   } else {
    //     void logout();
    //   }
    // }
  }, [isLoggedIn, isSignup, logout]);

  useEffect(() => {
    if (!isLoggedIn) return;
    const fetchData = async () => {
      if (siteId && siteId !== lastFetchedSiteIdRef.current) {
        const activeSite = sites.find((s) => s.id == siteId);
        initSite(activeSite ?? {});
        lastFetchedSiteIdRef.current = activeSite?.id;
        // Replay-only build: saved searches disabled (kept for later).
        // await searchStore.fetchSavedSearchList();
      }
    };

    void fetchData();
  }, [siteId, isLoggedIn]);

  const lastFetchedSiteIdRef = useRef<any>(null);

  function usePrevious(value: any) {
    const ref = useRef<any>(undefined);
    useEffect(() => {
      ref.current = value;
    }, [value]);
    return ref.current;
  }

  const prevIsLoggedIn = usePrevious(isLoggedIn);
  const previousLocation = usePrevious(location);

  const hideHeader =
    (location.pathname && location.pathname.includes('/session/')) ||
    // Replay-only build: disabled products kept for later.
    // location.pathname.includes('/assist/') ||
    // location.pathname.includes('multiview') ||
    // location.pathname.includes('/view-spot/') ||
    // location.pathname.includes('/spots/');
    false;
  if (isIframe) {
    return (
      <IFrameRoutes isLoggedIn={isLoggedIn} loading={loading} />
    );
  }

  return isLoggedIn ? (
    <NewModalProvider>
      <ModalProvider>
        <Loader loading={loading} className="flex-1">
          <Tracker />
          <Layout hideHeader={hideHeader}>
            <PrivateRoutes />
          </Layout>
        </Loader>
      </ModalProvider>
    </NewModalProvider>
  ) : (
    <PublicRoutes />
  );
};

export default withRouter(observer(Router));
