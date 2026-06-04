import { makeAutoObservable } from 'mobx';
import { tagWatchService } from 'App/services';
import { CreateTag, Tag } from 'App/services/TagWatchService';
import { projectStore } from '@/mstore';

export default class TagWatchStore {
  tags: Tag[] = [];

  isLoading = false;

  constructor() {
    makeAutoObservable(this);
  }

  setTags = (tags: Tag[]) => {
    this.tags = tags;
  };

  setLoading = (loading: boolean) => {
    this.isLoading = loading;
  };

  getTags = async (projectId?: number) => {
    // Replay-only build: tag watch disabled (kept for later).
    // if (this.isLoading) {
    //   return;
    // }
    // this.setLoading(true);
    // try {
    //   const pid = projectId || projectStore.active?.projectId;
    //   const tags: Tag[] = await tagWatchService.getTags(pid!);
    //   this.setTags(tags);
    //   return tags;
    // } catch (e) {
    //   console.error(e);
    // } finally {
    //   this.setLoading(false);
    // }
    void projectId;
    this.setTags([]);
    return [];
  };

  createTag = async (data: CreateTag, projectId?: number) => {
    // Replay-only build: tag watch disabled (kept for later).
    // try {
    //   const pid = projectId || projectStore.active?.projectId;
    //   const tagId: number = await tagWatchService.createTag(pid!, data);
    //   return tagId;
    // } catch (e) {
    //   console.error(e);
    // }
    void data;
    void projectId;
  };

  deleteTag = async (id: number, projectId?: number) => {
    // Replay-only build: tag watch disabled (kept for later).
    // try {
    //   const pid = projectId || projectStore.active?.projectId;
    //   await tagWatchService.deleteTag(pid!, id);
    //   this.setTags(this.tags.filter((t) => t.tagId !== id));
    // } catch (e) {
    //   console.error(e);
    // }
    void id;
    void projectId;
  };

  updateTagName = async (id: number, name: string, projectId?: number) => {
    // Replay-only build: tag watch disabled (kept for later).
    // try {
    //   const pid = projectId || projectStore.active?.projectId;
    //   await tagWatchService.updateTagName(pid!, id, name);
    //   const updatedTag = this.tags.find((t) => t.tagId === id);
    //   if (updatedTag) {
    //     this.setTags(
    //       this.tags.map((t) => (t.tagId === id ? { ...updatedTag, name } : t)),
    //     );
    //   }
    // } catch (e) {
    //   console.error(e);
    // }
    void id;
    void name;
    void projectId;
  };
}
