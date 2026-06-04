import React from 'react';

import APIClient from 'App/api_client';
import { services } from 'App/services';

import AiFiltersStore from './aiFiltersStore';
import AiSummaryStore from './aiSummaryStore';
import AlertStore from './alertsStore';
import AssistMultiviewStore from './assistMultiviewStore';
import AuditStore from './auditStore';
import CustomFieldStore from './customFieldStore';
import DashboardStore from './dashboardStore';
import ErrorStore from './errorStore';
import FilterStore from './filterStore';
import FunnelStore from './funnelStore';
import { IntegrationsStore } from './integrationsStore';
import IssueReportingStore from './issueReportingStore';
import LoginStore from './loginStore';
import MetricStore from './metricStore';
import NotesStore from './notesStore';
import NotificationStore from './notificationStore';
import ProjectsStore from './projectsStore';
import RecordingsStore from './recordingsStore';
import RoleStore from './roleStore';
import SearchStore from './searchStore';
import SearchStoreLive from './searchStoreLive';
import SessionStore from './sessionStore';
import SettingsStore from './settingsStore';
import TagWatchStore from './tagWatchStore';
import UiPlayerStore from './uiPlayerStore';
import userStore from './userStore';
import WeeklyReportStore from './weeklyReportConfigStore';
import logger from '@/logger';
import BillingStore from "@/mstore/billingStore";
import ClipStore from "@/mstore/clipStore";
import AnalyticsStore from "@/mstore/AnalyticsStore";

const projectStore = new ProjectsStore();
const sessionStore = new SessionStore();
const searchStore = new SearchStore();
const searchStoreLive = new SearchStoreLive();
const settingsStore = new SettingsStore();
const filterStore = new FilterStore();

const client = new APIClient();

export class RootStore {
  dashboardStore: DashboardStore;
  metricStore: MetricStore;
  funnelStore: FunnelStore;
  settingsStore: SettingsStore;
  userStore: typeof userStore;
  roleStore: RoleStore;
  auditStore: AuditStore;
  errorStore: ErrorStore;
  notificationStore: NotificationStore;
  sessionStore: SessionStore;
  notesStore: NotesStore;
  recordingsStore: RecordingsStore;
  assistMultiviewStore: AssistMultiviewStore;
  weeklyReportStore: WeeklyReportStore;
  alertsStore: AlertStore;
  tagWatchStore: TagWatchStore;
  aiSummaryStore: AiSummaryStore;
  aiFiltersStore: AiFiltersStore;
  loginStore: LoginStore;
  filterStore: FilterStore;
  uiPlayerStore: UiPlayerStore;
  issueReportingStore: IssueReportingStore;
  customFieldStore: CustomFieldStore;
  searchStore: SearchStore;
  searchStoreLive: SearchStoreLive;
  integrationsStore: IntegrationsStore;
  projectsStore: ProjectsStore;
  billingStore: BillingStore;
  clipStore: ClipStore;
  analyticsStore: AnalyticsStore;

  constructor() {
    this.dashboardStore = new DashboardStore();
    this.metricStore = new MetricStore();
    this.funnelStore = new FunnelStore();
    this.settingsStore = settingsStore;
    this.userStore = userStore;
    this.roleStore = new RoleStore();
    this.auditStore = new AuditStore();
    this.errorStore = new ErrorStore();
    this.notificationStore = new NotificationStore();
    this.sessionStore = sessionStore;
    this.notesStore = new NotesStore();
    this.recordingsStore = new RecordingsStore();
    this.assistMultiviewStore = new AssistMultiviewStore();
    this.weeklyReportStore = new WeeklyReportStore();
    this.alertsStore = new AlertStore();
    this.tagWatchStore = new TagWatchStore();
    this.aiSummaryStore = new AiSummaryStore();
    this.aiFiltersStore = new AiFiltersStore();
    // Replay-only build: Spot disabled (kept for later).
    // this.spotStore = new SpotStore();
    this.loginStore = new LoginStore();
    this.filterStore = filterStore;
    this.uiPlayerStore = new UiPlayerStore();
    this.issueReportingStore = new IssueReportingStore();
    this.customFieldStore = new CustomFieldStore();
    this.projectsStore = projectStore;
    this.searchStore = searchStore;
    this.searchStoreLive = searchStoreLive;
    this.integrationsStore = new IntegrationsStore();
    this.billingStore = new BillingStore();
    this.clipStore = new ClipStore();
    this.analyticsStore = new AnalyticsStore();
  }

  initClient() {
    client.setSiteIdCheck(projectStore.getSiteId);
    client.setJwt(userStore.getJwt());
    client.setJwtChecker(userStore.getJwt);
    client.setOnUpdateJwt(userStore.updateJwt);
    services.forEach((service) => {
      service.initClient(client);
    });
  }
}

const StoreContext = React.createContext<RootStore>({} as RootStore);

export function StoreProvider({ children, store }: any) {
  return (
    <StoreContext.Provider value={store}>{children}</StoreContext.Provider>
  );
}

export const useStore = () => React.useContext(StoreContext);

export const withStore = (Component: any) =>
  function (props: any) {
    return <Component {...props} mstore={useStore()} />;
  };

export {
  userStore,
  sessionStore,
  searchStore,
  searchStoreLive,
  projectStore,
  client,
  settingsStore,
  filterStore,
};
