import Period, { CUSTOM_RANGE } from 'Types/app/period';
import { FilterCategory, FilterKey } from 'Types/filter/filterType';
import {
  filtersMap,
  generateFilterOptions,
  liveFiltersMap,
} from 'Types/filter/newFilter';
import { makeAutoObservable, runInAction } from 'mobx';
import { searchService, sessionService } from 'App/services';
import Search from 'App/mstore/types/search';
import { checkFilterValue } from 'App/mstore/types/filter';
import FilterItem from 'App/mstore/types/filterItem';
import { filterStore, sessionStore, settingsStore } from 'App/mstore';
import SavedSearch, { ISavedSearch } from 'App/mstore/types/savedSearch';
import { iTag } from '@/services/NotesService';
import { Filter } from '@/mstore/types/filterConstants';

const PER_PAGE = 10;

export const checkValues = (key: any, value: any) => {
  if (key === FilterKey.DURATION) {
    return value[0] === '' || value[0] === null ? [0, value[1]] : value;
  }
  return value.filter((i: any) => i !== '' && i !== null);
};

export const filterMap = (filter: any) => {
  const {
    category,
    value,
    key,
    name,
    operator,
    sourceOperator,
    source,
    custom,
    isEvent,
    filters,
    dataType,
    propertyOrder,
    autoCaptured,
  } = filter;

  return {
    name: name || key,
    type:
      name || (category === FilterCategory.METADATA ? FilterKey.METADATA : key),
    value: checkValues(key, value),
    operator: operator || 'is',
    dataType: dataType || 'string',
    propertyOrder: propertyOrder || (isEvent ? 'then' : undefined),
    custom,
    source:
      category === FilterCategory.METADATA ? key?.replace(/^_/, '') : source,
    sourceOperator,
    isEvent: Boolean(isEvent),
    autoCaptured: Boolean(autoCaptured),
    filters: filters ? filters.map(filterMap) : [],
  };
};

export const TAB_MAP: any = {
  all: { name: 'All', type: 'all' },
  sessions: { name: 'Sessions', type: 'sessions' },
  bookmarks: { name: 'Bookmarks', type: 'bookmarks' },
  notes: { name: 'Notes', type: 'notes' },
  recommendations: { name: 'Recommendations', type: 'recommendations' },
};

class SearchStore {
  list: SavedSearch[] = [];
  latestRequestTime: number | null = null;
  latestList = [];
  alertMetricId: number | null = null;
  instance = new Search({
    startDate: Date.now() - 24 * 60 * 60 * 1000,
    endDate: Date.now(),
    filters: [],
    sort: 'startTs',
    order: 'desc',
    viewed: false,
    consoleLevel: '',
    eventsOrder: 'then',
  });
  savedSearch: ISavedSearch = new SavedSearch();
  filterSearchList: any = {};
  currentPage = 1;
  pageSize = PER_PAGE;
  activeTab = { name: 'All', type: 'all' };
  scrollY = 0;
  sessions = [];
  total: number = 0;
  latestSessionCount: number = 0;
  loadingFilterSearch = false;
  isSaving: boolean = false;
  activeTags: any[] = [];
  urlParsed: boolean = false;
  searchInProgress = false;
  savedSearchPage = 1;
  savedSearchPageSize = 100;
  savedSearchTotal = 0;

  constructor() {
    makeAutoObservable(this);
  }

  resetFilters = () => {
    this.instance = new Search({
      startDate: Date.now() - 24 * 60 * 60 * 1000,
      endDate: Date.now(),
      filters: [],
      sort: 'startTs',
      order: 'desc',
      viewed: false,
      consoleLevel: '',
      eventsOrder: 'then',
    });
  };

  setUrlParsed() {
    this.urlParsed = true;
  }

  get filterList() {
    return generateFilterOptions(filtersMap);
  }

  get filterListMobile() {
    return generateFilterOptions(filtersMap, true);
  }

  get filterListLive() {
    return generateFilterOptions(liveFiltersMap);
  }

  applySavedSearch(savedSearch: ISavedSearch) {
    this.savedSearch = new SavedSearch(savedSearch);

    const filtersData =
      savedSearch.data?.filters || savedSearch.filter?.filters || [];
    const searchData = savedSearch.data;
    const filters = filterStore.processFiltersFromData(filtersData);

    this.edit({
      filters: filters as any,
      startDate: searchData?.startTimestamp || this.instance.startDate,
      endDate: searchData?.endTimestamp || this.instance.endDate,
      sort: searchData?.sort || this.instance.sort,
      order: searchData?.order || this.instance.order,
      eventsOrder: searchData?.eventsOrder || this.instance.eventsOrder,
    });
    this.currentPage = 1;
  }

  async fetchSavedSearchList(page: number = 1, limit: number = 100) {
    // Replay-only build: saved searches disabled (kept for later).
    // const offset = (page - 1) * limit;
    // const response = await searchService.fetchSavedSearch({ limit, offset });
    // runInAction(() => {
    //   this.list =
    //     response.data?.map((item: any) => new SavedSearch(item)) || [];
    //   this.savedSearchTotal = response.total;
    //   this.savedSearchPage = page;
    //   this.savedSearchPageSize = limit;
    // });
    runInAction(() => {
      this.list = [];
      this.savedSearchTotal = 0;
      this.savedSearchPage = page;
      this.savedSearchPageSize = limit;
    });
  }

  changeSavedSearchPage(page: number) {
    this.savedSearchPage = page;
    void this.fetchSavedSearchList(page, this.savedSearchPageSize);
  }

  async loadSharedSearch(searchId: string): Promise<void> {
    // Replay-only build: saved searches disabled (kept for later).
    // try {
    //   const response = await searchService.getSavedSearch(searchId);
    //   if (response) {
    //     const filtersData =
    //       response.data?.filters || response.filter?.filters || [];
    //     const searchData = response.data;
    //     const filters = filterStore.processFiltersFromData(filtersData);
    //
    //     this.edit({
    //       filters: filters as any,
    //       startDate: searchData?.startTimestamp || this.instance.startDate,
    //       endDate: searchData?.endTimestamp || this.instance.endDate,
    //       sort: searchData?.sort || this.instance.sort,
    //       order: searchData?.order || this.instance.order,
    //       eventsOrder: searchData?.eventsOrder || this.instance.eventsOrder,
    //     });
    //
    //     this.currentPage = 1;
    //     await this.fetchSessions(true);
    //   }
    // } catch (error) {
    //   console.error('Failed to load shared search:', error);
    //   throw error;
    // }
    void searchId;
    return;
  }

  edit(instance: Partial<Search>) {
    this.instance = new Search({ ...this.instance, ...instance });
    this.currentPage = 1;
  }

  editSavedSearch(instance: Partial<SavedSearch>) {
    this.savedSearch = new SavedSearch(
      Object.assign(this.savedSearch.toData(), instance),
    );
  }

  apply(filter: any, fromUrl: boolean) {
    if (fromUrl) {
      this.instance = new Search(filter);
    } else {
      this.instance = new Search({ ...this.instance.toData(), ...filter });
    }
    this.currentPage = 1;
  }

  applyFilter(filter: any) {
    this.apply(filter, false);
  }

  fetchFilterSearch(params: any) {
    this.loadingFilterSearch = true;

    searchService
      .fetchFilterSearch(params)
      .then((response: any[]) => {
        this.filterSearchList = response.reduce(
          (
            acc: Record<string, { projectId: number; value: string }[]>,
            item: any,
          ) => {
            const { projectId, type, value } = item;
            if (!acc[type]) acc[type] = [];
            acc[type].push({ projectId, value });
            return acc;
          },
          {},
        );
      })
      .catch((error: any) => {
        console.error('Error fetching filter search:', error);
      })
      .finally(() => {
        this.loadingFilterSearch = false;
      });
  }

  updateCurrentPage(page: number, force = false) {
    this.currentPage = page;
    void this.fetchSessions(force);
  }

  nextPage = () => {
    this.currentPage += 1;
    return this.currentPage;
  };

  updateLatestSessionCount(count: number = 0) {
    this.latestSessionCount = count;
  }

  setActiveTab(tab: string) {
    runInAction(() => {
      this.activeTab = TAB_MAP[tab];
    });
  }

  resetTags = () => {
    this.activeTags = ['all'];
  };

  toggleTag(tag?: iTag) {
    // Replay-only build: tags/notes/bookmarks are disabled (kept for later).
    void tag;
    this.activeTags = ['all'];
  }

  async removeSavedSearch(id: string): Promise<void> {
    // Replay-only build: saved searches disabled (kept for later).
    // await searchService.deleteSavedSearch(id);
    // this.savedSearch = new SavedSearch({});
    // await this.fetchSavedSearchList();
    void id;
    runInAction(() => {
      this.savedSearch = new SavedSearch({});
      this.list = [];
      this.savedSearchTotal = 0;
    });
  }

  async saveAsShare(): Promise<void> {
    // Replay-only build: saved searches disabled (kept for later).
    // const searchData = this.instance.toSearch();
    // const ensureFilterFields = (filter: any): any => {
    //   return {
    //     ...filter,
    //     name: filter.name || filter.type,
    //     type: filter.type || filter.name,
    //     dataType: filter.dataType || 'string',
    //     operator: filter.operator || 'is',
    //     propertyOrder:
    //       filter.propertyOrder || (filter.isEvent ? 'then' : 'and'),
    //     filters: Array.isArray(filter.filters)
    //       ? filter.filters.map(ensureFilterFields)
    //       : [],
    //   };
    // };
    // const payload: any = {
    //   name: null,
    //   isPublic: false,
    //   isShare: true,
    //   data: {
    //     filters: searchData.filters.map(ensureFilterFields),
    //     startTimestamp: searchData.startTimestamp,
    //     endTimestamp: searchData.endTimestamp,
    //     sort: searchData.sort || 'startTs',
    //     order: searchData.order || 'desc',
    //     eventsOrder: searchData.eventsOrder || 'then',
    //     limit: searchData.limit || 10,
    //     page: searchData.page || 1,
    //   },
    // };
    // const savedSearchResponse = await searchService.saveSavedSearch(payload);
    // if (savedSearchResponse) {
    //   this.savedSearch = new SavedSearch({
    //     searchId: savedSearchResponse.searchId,
    //     isPublic: false,
    //     isShare: true,
    //     data: payload.data,
    //   });
    // }
    runInAction(() => {
      this.savedSearch = new SavedSearch({});
    });
  }

  async save(id?: string | null, rename = false): Promise<void> {
    // Replay-only build: saved searches disabled (kept for later).
    // const searchData = this.instance.toSearch();
    // const instance = this.savedSearch.toData();
    // const searchId = id || this.savedSearch.searchId;
    // const isNew = !searchId;
    // const filtersToSave = rename
    //   ? instance.data?.filters || instance.filter?.filters || []
    //   : searchData.filters;
    // const ensureFilterFields = (filter: any): any => {
    //   return {
    //     ...filter,
    //     name: filter.name || filter.type,
    //     type: filter.type || filter.name,
    //     dataType: filter.dataType || 'string',
    //     operator: filter.operator || 'is',
    //     propertyOrder:
    //       filter.propertyOrder || (filter.isEvent ? 'then' : 'and'),
    //     filters: Array.isArray(filter.filters)
    //       ? filter.filters.map(ensureFilterFields)
    //       : [],
    //   };
    // };
    // const payload: any = {
    //   name: instance.name || null,
    //   isPublic: instance.isPublic || false,
    //   isShare: instance.isShare || false,
    //   data: {
    //     filters: filtersToSave.map(ensureFilterFields),
    //     startTimestamp: searchData.startTimestamp,
    //     endTimestamp: searchData.endTimestamp,
    //     sort: searchData.sort || 'startTs',
    //     order: searchData.order || 'desc',
    //     eventsOrder: searchData.eventsOrder || 'then',
    //     limit: searchData.limit || 10,
    //     page: searchData.page || 1,
    //   },
    // };
    // let savedSearchResponse;
    // if (isNew) {
    //   savedSearchResponse = await searchService.saveSavedSearch(payload);
    // } else {
    //   savedSearchResponse = await searchService.updateSavedSearch(
    //     searchId!,
    //     payload,
    //   );
    // }
    // if (savedSearchResponse) {
    //   this.savedSearch = new SavedSearch({
    //     ...instance,
    //     searchId: savedSearchResponse.searchId || searchId,
    //     name: instance.name,
    //     isPublic: instance.isPublic,
    //     isShare: instance.isShare,
    //     data: payload.data,
    //   });
    // }
    // await this.fetchSavedSearchList();
    void id;
    void rename;
    runInAction(() => {
      this.savedSearch = new SavedSearch({});
      this.list = [];
      this.savedSearchTotal = 0;
    });
  }

  clearList() {
    this.list = [];
  }

  clearSearch() {
    const { instance } = this;
    this.edit(
      new Search({
        rangeValue: instance.rangeValue,
        startDate: instance.startDate,
        endDate: instance.endDate,
        filters: [],
      }),
    );

    this.savedSearch = new SavedSearch({});
    sessionStore.clearList();
  }

  async checkForLatestSessionCount(): Promise<void> {
    try {
      let filter = this.instance.toSearch();
      filter = this.applyTagFilter(filter, this.activeTags);
      filter = this.applyDurationFilter(filter);

      // Set time filter if we have the latest request time
      if (this.latestRequestTime) {
        const period = Period({
          rangeName: CUSTOM_RANGE,
          start: this.latestRequestTime,
          end: Date.now(),
        });
        const timeRange: any = period.toJSON();
        filter.startDate = timeRange.startDate;
        filter.endDate = timeRange.endDate;
      }

      // Only need the total count, not actual records
      filter.limit = 1;
      filter.page = 1;

      const response = await sessionService.getSessions(filter);

      runInAction(() => {
        if (response?.total && response.total > sessionStore.total) {
          this.latestSessionCount = response.total - sessionStore.total;
        } else {
          this.latestSessionCount = 0;
        }
      });
    } catch (error) {
      console.error('Failed to check for latest session count:', error);
    }
  }

  addFilterOnce = (filter: any) => {
    const isPresent = this.instance.filters.find(
      (f) => f.name === filter.name && f.isEvent === filter.isEvent,
    );
    if (isPresent) {
      return;
    }
    this.addFilter(filter);
  };

  addFilter = (filter: any) => {
    if (filter.isEvent && (!filter.filters || filter.filters.length === 0)) {
      filterStore.getEventFilters(filter.id).then((props) => {
        filter.filters = props?.filter((prop) => prop.defaultProperty);
      });
    }

    if (!filter.isEvent) {
      const isPresent = this.instance.filters.find(
        (f) => f.name === filter.name && f.isEvent === false,
      );
      if (isPresent) {
        return;
      }
    }

    filter.value = checkFilterValue(filter.value);
    filter.operator = filter.operator || 'is';
    filter.filters = filter.filters
      ? filter.filters.map((subFilter: any) => ({
          ...subFilter,
          value: checkFilterValue(subFilter.value),
        }))
      : null;
    const inst = this.instance.toData();
    inst.filters.push(filter);
    this.instance = new Search(inst);
    this.currentPage = 1;

    if (filter.value && filter.value[0] && filter.value[0] !== '') {
      void this.fetchSessions();
    }
  }

  moveFilter(draggedIndex: number, newPosition: number) {
    const newFilters = this.instance.filters.slice();
    const [removed] = newFilters.splice(draggedIndex, 1);
    newFilters.splice(newPosition, 0, removed);

    this.instance = new Search({
      ...this.instance.toData(),
      filters: newFilters,
    });
  }

  addFilterByKeyAndValue(
    key: any,
    value: any,
    operator?: string,
    sourceOperator?: string,
    source?: string,
  ) {
    const defaultFilter = { ...filtersMap[key] };

    defaultFilter.value = value;

    if (operator) {
      defaultFilter.operator = operator;
    }
    if (defaultFilter.hasSource && source && sourceOperator) {
      defaultFilter.sourceOperator = sourceOperator;
      defaultFilter.source = source;
    }

    this.addFilter(defaultFilter);
  }

  updateSearch = (search: Partial<Search>) => {
    this.instance = Object.assign(this.instance, search);
  };

  updateFilter = (index: number, search: Partial<Filter>) => {
    const newFilters = [...this.instance.filters];
    newFilters[index] = {
      ...newFilters[index],
      ...search,
    };

    this.instance = new Search({
      ...this.instance.toData(),
      filters: newFilters,
    });
  };

  removeFilter = (index: number) => {
    const newFilters = [...this.instance.filters];
    newFilters.splice(index, 1);

    this.instance = new Search({
      ...this.instance.toData(),
      filters: newFilters,
    });
  };

  setScrollPosition = (y: number) => {
    this.scrollY = y;
  };

  async fetchSessions(
    force: boolean = false,
    bookmarked: boolean = false,
  ): Promise<void> {
    if (this.searchInProgress) return;

    let filter = this.instance.toSearch();
    filter = this.applyTagFilter(filter, this.activeTags);
    filter = this.applyDurationFilter(filter);
    this.latestRequestTime = filter.startDate;
    this.latestList = [];
    this.searchInProgress = true;
    await sessionStore
      .fetchSessions(
        {
          ...filter,
          page: this.currentPage,
          limit: this.pageSize,
          bookmarked: bookmarked ? true : undefined,
        },
        force,
      )
      .finally(() => {
        this.searchInProgress = false;
      });
  }

  private applyTagFilter(filter: any, activeTags: string[]): any {
    // Replay-only build: tags/notes/bookmarks are disabled (kept for later).
    // if (!activeTags?.length || activeTags[0] === 'all') {
    //   return filter;
    // }
    //
    // const tagFilter = filterStore.findEvent({
    //   name: 'issue',
    //   isEvent: false,
    //   autoCaptured: true,
    // });
    //
    // if (!tagFilter) {
    //   console.error('Tag filter not found for issue');
    //   return filter;
    // }
    //
    // tagFilter.value = [activeTags[0]];
    //
    // return {
    //   ...filter,
    //   filters: [...filter.filters, tagFilter.toJson()],
    // };
    void activeTags;
    return filter;
  }

  private applyDurationFilter(filter: any): any {
    if (filter.filters.some((f: any) => f.name === FilterKey.DURATION)) {
      return filter;
    }

    const { durationFilter } = settingsStore.sessionSettings;

    if (!durationFilter?.count || durationFilter.count <= 0) {
      return filter;
    }

    const multiplier = durationFilter.countType === 'sec' ? 1000 : 60000;
    const amount = durationFilter.count * multiplier;
    const value =
      durationFilter.operator === '<'
        ? [amount.toString()]
        : [null, amount.toString()];

    const durationFilterConfig = {
      autoCaptured: true,
      dataType: 'number',
      name: FilterKey.DURATION,
      value,
      operator: '=',
    };

    return {
      ...filter,
      filters: [...filter.filters, durationFilterConfig],
    };
  }
}

export default SearchStore;
