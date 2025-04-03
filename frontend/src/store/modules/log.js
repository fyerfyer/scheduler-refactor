import logService from '@/services/logService';

export default {
    namespaced: true,

    state: {
        logs: [],
        currentLog: null,
        jobStats: null,
        totalCount: 0,
        loading: false,
        error: null
    },

    mutations: {
        SET_LOGS(state, { logs, total }) {
            state.logs = logs;
            state.totalCount = total;
        },
        SET_CURRENT_LOG(state, log) {
            state.currentLog = log;
        },
        SET_JOB_STATS(state, stats) {
            state.jobStats = stats;
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
        // Fetch logs with optional job filtering and pagination
        async fetchLogs({ commit }, { jobName = '', page = 1, pageSize = 10 } = {}) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                const data = await logService.listLogs(jobName, page, pageSize);
                commit('SET_LOGS', {
                    logs: data.logs || [],
                    total: data.total || 0
                });
                return data;
            } catch (error) {
                commit('SET_ERROR', error.message || 'Failed to fetch logs');
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Fetch the latest log for a specific job
        async fetchJobLog({ commit }, jobName) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                const log = await logService.getJobLog(jobName);
                commit('SET_CURRENT_LOG', log);
                return log;
            } catch (error) {
                commit('SET_ERROR', error.message || `Failed to fetch log for job ${jobName}`);
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Fetch a specific log by job name and timestamp
        async fetchLogByTimestamp({ commit }, { jobName, timestamp }) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                const log = await logService.getLogByTimestamp(jobName, timestamp);
                commit('SET_CURRENT_LOG', log);
                return log;
            } catch (error) {
                commit('SET_ERROR', error.message || `Failed to fetch log for job ${jobName}`);
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Fetch statistics for a specific job
        async fetchJobStats({ commit }, { jobName, days = 7 }) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                const stats = await logService.getJobStats(jobName, days);
                commit('SET_JOB_STATS', stats);
                return stats;
            } catch (error) {
                commit('SET_ERROR', error.message || `Failed to fetch stats for job ${jobName}`);
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Fetch overall log statistics
        async fetchAllStats({ /* commit */ }, { days = 1 } = {}) {
            try {
                return await logService.getAllStats(days);
            } catch (error) {
                console.error('Error fetching all stats:', error);
                return null;
            }
        },

        // Clear current log
        clearCurrentLog({ commit }) {
            commit('SET_CURRENT_LOG', null);
        }
    },

    getters: {
        // Get logs for a specific job
        getLogsByJob: (state) => (jobName) => {
            return state.logs.filter(log => log.jobName === jobName);
        },

        // Calculate success rate from job stats
        successRate: (state) => {
            if (!state.jobStats || state.jobStats.totalCount === 0) {
                return 0;
            }
            return Math.round((state.jobStats.successCount / state.jobStats.totalCount) * 100);
        }
    }
};