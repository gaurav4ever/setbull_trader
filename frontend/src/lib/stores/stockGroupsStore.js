import { writable } from 'svelte/store';
import * as api from '../services/stockGroupService.js';

function createStockGroupsStore() {
    const { subscribe, set, update } = writable({
        groups: [],
        loading: false,
        error: ''
    });

    async function loadGroups() {
        set({ groups: [], loading: true, error: '' });
        try {
            const groups = await api.listGroups();
            set({ groups, loading: false, error: '' });
        } catch (e) {
            set({ groups: [], loading: false, error: e.message || 'Failed to load groups' });
        }
    }

    /**
     * @param {string} entryType
     * @param {string[]} stockIds
     */
    async function createGroup(entryType, stockIds) {
        update(state => ({ ...state, loading: true, error: '' }));
        try {
            await api.createGroup(entryType, stockIds);
            await loadGroups();
        } catch (e) {
            update(state => ({ ...state, loading: false, error: e.message || 'Failed to create group' }));
            throw e;
        }
    }

    return {
        subscribe,
        loadGroups,
        createGroup
    };
}

export const stockGroupsStore = createStockGroupsStore(); 