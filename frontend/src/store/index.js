import { createStore } from 'vuex';

// Import modules
import job from './modules/job';
import log from './modules/log';
import worker from './modules/worker';

export default createStore({
    modules: {
        job,
        log,
        worker
    },
    // Global state, if needed
    state: {},
    // Global mutations, if needed
    mutations: {},
    // Global actions, if needed
    actions: {},
    // Global getters, if needed
    getters: {}
});