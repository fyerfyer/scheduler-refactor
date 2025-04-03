<template>
  <form @submit.prevent="submitForm">
    <div class="mb-3">
      <label for="jobName" class="form-label">Job Name*</label>
      <input
        type="text"
        class="form-control"
        id="jobName"
        v-model="job.name"
        :disabled="isEditing"
        required
      >
      <div class="form-text text-muted">Job name must be unique and cannot be changed after creation.</div>
    </div>

    <div class="mb-3">
      <label for="command" class="form-label">Command*</label>
      <textarea
        class="form-control"
        id="command"
        v-model="job.command"
        rows="3"
        required
      ></textarea>
      <div class="form-text text-muted">Shell command to execute.</div>
    </div>

    <div class="mb-3">
      <label for="cronExpr" class="form-label">Cron Expression*</label>
      <input
        type="text"
        class="form-control"
        id="cronExpr"
        v-model="job.cronExpr"
        required
      >
      <div class="form-text text-muted">For example: "0 0 * * *" (run at midnight every day).</div>
    </div>

    <div class="mb-3">
      <label for="timeout" class="form-label">Timeout (seconds)</label>
      <input
        type="number"
        class="form-control"
        id="timeout"
        v-model.number="job.timeout"
        min="0"
      >
      <div class="form-text text-muted">Maximum time allowed for job execution. 0 means no timeout.</div>
    </div>

    <div class="mb-3 form-check">
      <input
        type="checkbox"
        class="form-check-input"
        id="disabled"
        v-model="job.disabled"
      >
      <label class="form-check-label" for="disabled">Disabled</label>
    </div>

    <div class="d-flex justify-content-between">
      <button type="button" class="btn btn-secondary" @click="cancel">Cancel</button>
      <button type="submit" class="btn btn-primary">{{ isEditing ? 'Update' : 'Create' }} Job</button>
    </div>
  </form>
</template>

<script>
export default {
  name: 'JobForm',
  props: {
    initialJob: {
      type: Object,
      default() {
        return {
          name: '',
          command: '',
          cronExpr: '',
          timeout: 60,
          disabled: false
        }
      }
    },
    isEditing: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      job: { ...this.initialJob }
    }
  },
  watch: {
    initialJob: {
      handler(newVal) {
        this.job = { ...newVal };
      },
      deep: true
    }
  },
  methods: {
    submitForm() {
      // Validate cron expression (basic validation)
      if (!this.validateCronExpression(this.job.cronExpr)) {
        alert('Invalid cron expression');
        return;
      }

      this.$emit('submit', { ...this.job });
    },
    cancel() {
      this.$emit('cancel');
    },
    validateCronExpression(expr) {
      // This is a very basic validation - in production, you'd want more robust validation
      const cronPattern = /^(\*|[0-9]+|\*\/[0-9]+|[0-9]+-[0-9]+)((\s+(\*|[0-9]+|\*\/[0-9]+|[0-9]+-[0-9]+)){4,6})$/;
      return cronPattern.test(expr);
    }
  }
}
</script>