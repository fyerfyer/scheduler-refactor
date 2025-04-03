import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import JobsView from '../views/JobsView.vue'
import JobCreateView from '../views/JobCreateView.vue'
import JobDetailView from '../views/JobDetailView.vue'
import LogsView from '../views/LogsView.vue'
import WorkersView from '../views/WorkersView.vue'

const routes = [
    {
        path: '/',
        name: 'dashboard',
        component: DashboardView
    },
    {
        path: '/jobs',
        name: 'jobs',
        component: JobsView
    },
    {
        path: '/jobs/create',
        name: 'job-create',
        component: JobCreateView
    },
    {
        path: '/jobs/edit/:name',
        name: 'job-edit',
        component: JobCreateView,
        props: true
    },
    {
        path: '/jobs/:name',
        name: 'job-detail',
        component: JobDetailView,
        props: true
    },
    {
        path: '/logs',
        name: 'logs',
        component: LogsView
    },
    {
        path: '/logs/:jobName/:timestamp?',
        name: 'log-detail',
        component: LogsView,
        props: true
    },
    {
        path: '/workers',
        name: 'workers',
        component: WorkersView
    }
]

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes
})

export default router