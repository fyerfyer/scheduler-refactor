import jobService from '@/services/jobService';

export default {
    namespaced: true,

    state: {
        jobs: [],
        currentJob: null,
        runningJobs: [],
        loading: false,
        saving: false,
        error: null
    },

    mutations: {
        SET_JOBS(state, jobs) {
            state.jobs = jobs;
        },
        SET_CURRENT_JOB(state, job) {
            state.currentJob = job;
        },
        SET_RUNNING_JOBS(state, runningJobs) {
            state.runningJobs = runningJobs;
        },
        SET_LOADING(state, isLoading) {
            state.loading = isLoading;
        },
        SET_SAVING(state, isSaving) {
            state.saving = isSaving;
        },
        SET_ERROR(state, error) {
            state.error = error;
        },
        CLEAR_ERROR(state) {
            state.error = null;
        }
    },

    actions: {
        // Fetch all jobs or search jobs by keyword
        async fetchJobs({ commit }, { keyword = '' } = {}) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                const jobs = await jobService.listJobs(keyword);
                commit('SET_JOBS', jobs);
                return jobs;
            } catch (error) {
                commit('SET_ERROR', error.message || 'Failed to fetch jobs');
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Fetch a specific job by name
        async fetchJob({ commit }, jobName) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                const job = await jobService.getJob(jobName);
                commit('SET_CURRENT_JOB', job);
                return job;
            } catch (error) {
                commit('SET_ERROR', error.message || `Failed to fetch job ${jobName}`);
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Save a job (create or update)
        async saveJob({ commit }, jobData) {
            commit('SET_SAVING', true);
            commit('CLEAR_ERROR');

            try {
                const result = await jobService.saveJob(jobData);
                return result;
            } catch (error) {
                commit('SET_ERROR', error.message || 'Failed to save job');
                throw error;
            } finally {
                commit('SET_SAVING', false);
            }
        },

        // Delete a job
        async deleteJob({ commit }, jobName) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                await jobService.deleteJob(jobName);
            } catch (error) {
                commit('SET_ERROR', error.message || `Failed to delete job ${jobName}`);
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Enable a job
        async enableJob({ commit }, jobName) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                await jobService.enableJob(jobName);
            } catch (error) {
                commit('SET_ERROR', error.message || `Failed to enable job ${jobName}`);
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Disable a job
        async disableJob({ commit }, jobName) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                await jobService.disableJob(jobName);
            } catch (error) {
                commit('SET_ERROR', error.message || `Failed to disable job ${jobName}`);
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Kill a running job
        async killJob({ commit }, jobName) {
            commit('SET_LOADING', true);
            commit('CLEAR_ERROR');

            try {
                await jobService.killJob(jobName);
            } catch (error) {
                commit('SET_ERROR', error.message || `Failed to kill job ${jobName}`);
                throw error;
            } finally {
                commit('SET_LOADING', false);
            }
        },

        // Get currently running jobs
        async fetchRunningJobs({ commit }) {
            try {
                const runningJobs = await jobService.getRunningJobs();
                commit('SET_RUNNING_JOBS', runningJobs);
                return runningJobs;
            } catch (error) {
                console.error('Error fetching running jobs:', error);
                // Don't set global error state for this background operation
                return [];
            }
        },

        // Clear current job
        clearCurrentJob({ commit }) {
            commit('SET_CURRENT_JOB', null);
        }
    },

    getters: {
        isJobRunning: (state) => (jobName) => {
            return state.runningJobs.includes(jobName);
        },
        getJobByName: (state) => (jobName) => {
            return state.jobs.find(job => job.name === jobName) || null;
        },
        enabledJobs: (state) => {
            return state.jobs.filter(job => !job.disabled);
        },
        disabledJobs: (state) => {
            return state.jobs.filter(job => job.disabled);
        }
    }
};