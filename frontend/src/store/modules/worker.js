import workerService from '@/services/workerService';

export default {
    namespaced: true,

    state: {
        workers: [],
        stats: {
            total: 0,
            online: 0,
            offline: 0,
            avgCpuUsage: 0,
            avgMemUsage: 0
        },
        loading: false,
        error: null
    },

    mutations: {
        SET_WORKERS(state, workers) {
            state.workers = workers;
        },
        SET_WORKER_STATS(state, stats) {
            state.stats = stats;
        },
        SET_LOADING(state, isLoading) {
            state.loading = isLoading;
        },
        SET_ERROR(state, error) {
            state.error = error;
        },
        CLEAR_ERROR(state) {
            state.error = null;
        }
    },

    actions: {
        // Fetch all worker nodes
        async fetchWorkers({ commit }) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                const workers = await workerService.listWorkers();
                commit('SET_WORKERS', workers);
                return workers;
            } catch (error) {
                commit('SET_ERROR', error.message || 'Failed to fetch workers');
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Fetch worker statistics
        async fetchWorkerStats({ commit }) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                const stats = await workerService.getWorkerStats();
                commit('SET_WORKER_STATS', stats);
                return stats;
            } catch (error) {
                commit('SET_ERROR', error.message || 'Failed to fetch worker statistics');
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        }
    },

    getters: {
        onlineWorkers: (state) => {
            return state.workers.filter(worker => worker.status === 'online');
        },
        offlineWorkers: (state) => {
            return state.workers.filter(worker => worker.status === 'offline');
        },
        getWorkerById: (state) => (id) => {
            return state.workers.find(worker => worker.ip === id) || null;
        }
    }
};