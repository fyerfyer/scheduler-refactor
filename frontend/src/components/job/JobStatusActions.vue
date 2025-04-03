<template>
  <div class="btn-group">
    <button
      class="btn btn-sm btn-outline-primary"
      @click="viewDetails"
      title="View Details"
    >
      <i class="bi bi-eye"></i>
    </button>

    <!-- Enable/Disable button -->
    <button
      class="btn btn-sm"
      :class="job.disabled ? 'btn-outline-success' : 'btn-outline-secondary'"
      @click="toggleStatus"
      :title="job.disabled ? 'Enable Job' : 'Disable Job'"
    >
      <i class="bi" :class="job.disabled ? 'bi-play-fill' : 'bi-pause-fill'"></i>
    </button>

    <!-- Kill button (only shown for running jobs) -->
    <button
      v-if="isRunning"
      class="btn btn-sm btn-outline-danger"
      @click="killJob"
      title="Kill Running Job"
    >
      <i class="bi bi-x-circle"></i>
    </button>

    <!-- Delete button -->
    <button
      class="btn btn-sm btn-outline-danger"
      @click="confirmDelete"
      title="Delete Job"
    >
      <i class="bi bi-trash"></i>
    </button>
  </div>
</template>

<script>
export default {
  name: 'JobStatusActions',
  props: {
    job: {
      type: Object,
      required: true
    },
    isRunning: {
      type: Boolean,
      default: false
    }
  },
  methods: {
    viewDetails() {
      this.$router.push(`/jobs/${this.job.name}`);
    },
    async toggleStatus() {
      try {
        const endpoint = this.job.disabled ?
          `/api/v1/job/enable/${this.job.name}` :
          `/api/v1/job/disable/${this.job.name}`;

        const response = await fetch(endpoint, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          }
        });

        const data = await response.json();

        if (data.code === 0) {
          // Success - emit event to refresh the job list
          this.$emit('reload');
        } else {
          alert(`Failed to ${this.job.disabled ? 'enable' : 'disable'} job: ${data.message}`);
        }
      } catch (error) {
        console.error('Error toggling job status:', error);
        alert('An error occurred while updating the job status');
      }
    },
    async killJob() {
      if (!confirm(`Are you sure you want to kill the running job "${this.job.name}"?`)) {
        return;
      }

      try {
        const response = await fetch(`/api/v1/job/kill/${this.job.name}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          }
        });

        const data = await response.json();

        if (data.code === 0) {
          // Success - emit event to refresh the job list
          this.$emit('reload');
        } else {
          alert(`Failed to kill job: ${data.message}`);
        }
      } catch (error) {
        console.error('Error killing job:', error);
        alert('An error occurred while killing the job');
      }
    },
    async confirmDelete() {
      if (!confirm(`Are you sure you want to delete the job "${this.job.name}"?`)) {
        return;
      }

      try {
        const response = await fetch(`/api/v1/job/${this.job.name}`, {
          method: 'DELETE'
        });

        const data = await response.json();

        if (data.code === 0) {
          // Success - emit event to refresh the job list
          this.$emit('reload');
        } else {
          alert(`Failed to delete job: ${data.message}`);
        }
      } catch (error) {
        console.error('Error deleting job:', error);
        alert('An error occurred while deleting the job');
      }
    }
  }
}
</script>