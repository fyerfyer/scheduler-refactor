<template>
  <div class="job-list">
    <!-- Search and filter -->
    <div class="mb-4 d-flex justify-content-between align-items-center">
      <div class="input-group w-50">
        <input
          type="text"
          class="form-control"
          placeholder="Search jobs..."
          v-model="searchKeyword"
          @keyup.enter="loadJobs"
        >
        <button class="btn btn-outline-primary" type="button" @click="loadJobs">
          <i class="bi bi-search"></i> Search
        </button>
      </div>

      <router-link to="/jobs/create" class="btn btn-success">
        <i class="bi bi-plus-circle"></i> Create Job
      </router-link>
    </div>

    <!-- Jobs table -->
    <div class="table-responsive">
      <table class="table table-hover">
        <thead class="table-light">
          <tr>
            <th>Name</th>
            <th>Command</th>
            <th>Schedule</th>
            <th>Status</th>
            <th>Last Updated</th>
            <th class="text-center">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading">
            <td colspan="6" class="text-center py-4">
              <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">Loading...</span>
              </div>
            </td>
          </tr>
          <tr v-else-if="jobs.length === 0">
            <td colspan="6" class="text-center py-4">
              No jobs found. <router-link to="/jobs/create">Create a new job</router-link>
            </td>
          </tr>
          <tr v-for="job in jobs" :key="job.name">
            <td>
              <router-link :to="`/jobs/${job.name}`" class="fw-bold text-decoration-none">
                {{ job.name }}
              </router-link>
            </td>
            <td>
              <span class="command-preview">{{ truncateCommand(job.command, 40) }}</span>
            </td>
            <td>{{ job.cronExpr }}</td>
            <td>
              <status-badge :status="job.disabled ? 'disabled' : 'enabled'" type="job" />
              <status-badge v-if="runningJobs.includes(job.name)" status="running" type="job" class="ms-1" />
            </td>
            <td>{{ formatDate(job.updatedAt) }}</td>
            <td class="text-center">
              <job-status-actions
                :job="job"
                :is-running="runningJobs.includes(job.name)"
                @reload="loadJobs"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <pagination
      v-if="totalJobs > 0"
      :current-page="currentPage"
      :total-items="totalJobs"
      :page-size="pageSize"
      @page-changed="onPageChange"
    />
  </div>
</template>

<script>
import StatusBadge from '../common/StatusBadge.vue';
import Pagination from '../common/Pagination.vue';
import JobStatusActions from './JobStatusActions.vue';

export default {
  name: 'JobList',
  components: {
    StatusBadge,
    Pagination,
    JobStatusActions
  },
  data() {
    return {
      jobs: [],
      loading: true,
      searchKeyword: '',
      currentPage: 1,
      pageSize: 10,
      totalJobs: 0,
      runningJobs: [] // List of currently running job names
    }
  },
  created() {
    this.loadJobs();
    // Poll for running jobs every 5 seconds
    this.pollInterval = setInterval(this.checkRunningJobs, 5000);
  },
  beforeUnmount() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
    }
  },
  methods: {
    loadJobs() {
      this.loading = true;

      // In a real app, you would call your API service here
      // Example: this.$store.dispatch('jobs/fetchJobs', { keyword: this.searchKeyword })

      // Simulating API call for now
      setTimeout(() => {
        // Replace with actual API call
        fetch(`/api/v1/job/list?keyword=${this.searchKeyword}`)
          .then(response => response.json())
          .then(data => {
            if (data.code === 0) {
              this.jobs = data.data;
              this.totalJobs = this.jobs.length;
              this.loading = false;
            } else {
              console.error('Failed to load jobs:', data.message);
              this.loading = false;
            }
          })
          .catch(error => {
            console.error('Error loading jobs:', error);
            this.loading = false;
          });
      }, 500);
    },
    checkRunningJobs() {
      // In a real app, you would call an API to get currently running jobs
      // This is just a placeholder
      fetch('/api/v1/job/running')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0) {
            this.runningJobs = data.data || [];
          }
        })
        .catch(error => {
          console.error('Error checking running jobs:', error);
        });
    },
    onPageChange(page) {
      this.currentPage = page;
      this.loadJobs();
    },
    truncateCommand(command, maxLength) {
      if (command.length <= maxLength) return command;
      return command.substring(0, maxLength) + '...';
    },
    formatDate(timestamp) {
      if (!timestamp) return 'N/A';
      const date = new Date(timestamp * 1000);
      return date.toLocaleString();
    }
  }
}
</script>

<style scoped>
.command-preview {
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: inline-block;
  max-width: 250px;
}
</style>