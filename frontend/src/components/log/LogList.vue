<template>
  <div class="log-list">
    <div class="mb-4 d-flex justify-content-between align-items-center">
      <div class="input-group w-50">
        <select v-model="selectedJob" class="form-select" @change="loadLogs">
          <option value="">All Jobs</option>
          <option v-for="job in jobs" :key="job" :value="job">{{ job }}</option>
        </select>
        <button class="btn btn-outline-primary" type="button" @click="loadLogs">
          <i class="bi bi-filter"></i> Filter
        </button>
      </div>

      <div>
        <button class="btn btn-outline-secondary" @click="refreshLogs">
          <i class="bi bi-arrow-clockwise"></i> Refresh
        </button>
      </div>
    </div>

    <div class="table-responsive">
      <table class="table table-hover">
        <thead class="table-light">
          <tr>
            <th>Job</th>
            <th>Status</th>
            <th>Start Time</th>
            <th>Duration</th>
            <th>Worker</th>
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
          <tr v-else-if="logs.length === 0">
            <td colspan="6" class="text-center py-4">
              No logs found. Try changing your filter settings.
            </td>
          </tr>
          <tr v-for="log in logs" :key="`${log.jobName}-${log.startTime}`">
            <td>{{ log.jobName }}</td>
            <td>
              <status-badge
                :status="getLogStatus(log)"
                type="execution"
              />
            </td>
            <td>{{ formatDate(log.startTime) }}</td>
            <td>{{ formatDuration(log.startTime, log.endTime) }}</td>
            <td>{{ log.workerIp }}</td>
            <td class="text-center">
              <button class="btn btn-sm btn-outline-primary" @click="viewLogDetails(log)">
                <i class="bi bi-file-text"></i> Details
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <pagination
      v-if="totalLogs > 0"
      :current-page="currentPage"
      :total-items="totalLogs"
      :page-size="pageSize"
      @page-changed="onPageChange"
    />
  </div>
</template>

<script>
import StatusBadge from '../common/StatusBadge.vue';
import Pagination from '../common/Pagination.vue';

export default {
  name: 'LogList',
  components: {
    StatusBadge,
    Pagination
  },
  data() {
    return {
      logs: [],
      jobs: [], // List of available job names
      selectedJob: '',
      loading: true,
      currentPage: 1,
      pageSize: 10,
      totalLogs: 0
    }
  },
  created() {
    this.loadJobs();
    this.loadLogs();
  },
  methods: {
    loadJobs() {
      // In a real app, this would be replaced with an actual API call
      fetch('/api/v1/job/list')
        .then(response => response.json())
        .then(data => {
          if (data.code === 0 && data.data) {
            // Extract unique job names
            this.jobs = [...new Set(data.data.map(job => job.name))];
          }
        })
        .catch(error => {
          console.error('Error loading jobs:', error);
        });
    },
    loadLogs() {
      this.loading = true;

      const params = new URLSearchParams({
        page: this.currentPage,
        pageSize: this.pageSize
      });

      if (this.selectedJob) {
        params.append('jobName', this.selectedJob);
      }

      // In a real app, this would be replaced with an actual API call
      fetch(`/api/v1/log/list?${params.toString()}`)
        .then(response => response.json())
        .then(data => {
          if (data.code === 0 && data.data) {
            this.logs = data.data.logs || [];
            this.totalLogs = data.data.total || 0;
          } else {
            this.logs = [];
            this.totalLogs = 0;
          }
          this.loading = false;
        })
        .catch(error => {
          console.error('Error loading logs:', error);
          this.loading = false;
        });
    },
    refreshLogs() {
      this.loadLogs();
    },
    onPageChange(page) {
      this.currentPage = page;
      this.loadLogs();
    },
    viewLogDetails(log) {
      // You could either navigate to a details page or show a modal
      this.$router.push(`/logs/${log.jobName}/${log.startTime}`);
    },
    formatDate(timestamp) {
      if (!timestamp) return 'N/A';
      const date = new Date(timestamp * 1000);
      return date.toLocaleString();
    },
    formatDuration(startTime, endTime) {
      if (!startTime || !endTime) return 'N/A';

      const durationSec = endTime - startTime;

      if (durationSec < 60) {
        return `${durationSec} sec`;
      } else if (durationSec < 3600) {
        const minutes = Math.floor(durationSec / 60);
        const seconds = durationSec % 60;
        return `${minutes} min ${seconds} sec`;
      } else {
        const hours = Math.floor(durationSec / 3600);
        const minutes = Math.floor((durationSec % 3600) / 60);
        return `${hours} hr ${minutes} min`;
      }
    },
    getLogStatus(log) {
      if (log.isTimeout) return 'timeout';
      if (log.exitCode === 0) return 'success';
      return 'error';
    }
  }
}
</script>