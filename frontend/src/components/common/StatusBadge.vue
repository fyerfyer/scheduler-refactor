<template>
  <span class="badge" :class="badgeClass">{{ statusText }}</span>
</template>

<script>
export default {
  name: 'StatusBadge',
  props: {
    status: {
      type: String,
      required: true
    },
    type: {
      type: String,
      default: 'default', // 'default', 'job', 'worker', 'execution'
      validator: (value) => {
        return ['default', 'job', 'worker', 'execution'].includes(value);
      }
    }
  },
  computed: {
    statusText() {
      if (this.type === 'job') {
        if (this.status === 'disabled') return 'Disabled';
        if (this.status === 'enabled') return 'Enabled';
        if (this.status === 'running') return 'Running';
      } else if (this.type === 'worker') {
        if (this.status === 'online') return 'Online';
        if (this.status === 'offline') return 'Offline';
      } else if (this.type === 'execution') {
        if (this.status === 'success') return 'Success';
        if (this.status === 'error') return 'Error';
        if (this.status === 'timeout') return 'Timeout';
        if (this.status === 'killed') return 'Killed';
      }

      // Default: just capitalize the status
      return this.status.charAt(0).toUpperCase() + this.status.slice(1);
    },
    badgeClass() {
      // Default color mapping
      const colorMap = {
        'default': 'bg-secondary',
        // Job statuses
        'disabled': 'bg-secondary',
        'enabled': 'bg-success',
        'running': 'bg-primary',
        // Worker statuses
        'online': 'bg-success',
        'offline': 'bg-danger',
        // Execution statuses
        'success': 'bg-success',
        'error': 'bg-danger',
        'timeout': 'bg-warning text-dark',
        'killed': 'bg-dark'
      };

      return colorMap[this.status] || 'bg-secondary';
    }
  }
}
</script>

<style scoped>
.badge {
  font-size: 0.8rem;
  padding: 0.35em 0.65em;
}
</style>