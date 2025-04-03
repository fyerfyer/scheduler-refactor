<template>
  <nav aria-label="Page navigation" v-if="totalPages > 1">
    <ul class="pagination justify-content-center">
      <!-- Previous button -->
      <li class="page-item" :class="{ disabled: currentPage === 1 }">
        <a class="page-link" href="#" @click.prevent="onPageChange(currentPage - 1)">
          <span aria-hidden="true">&laquo;</span>
        </a>
      </li>

      <!-- First page -->
      <li class="page-item" :class="{ active: currentPage === 1 }">
        <a class="page-link" href="#" @click.prevent="onPageChange(1)">1</a>
      </li>

      <!-- Ellipsis if needed -->
      <li class="page-item disabled" v-if="startPage > 2">
        <span class="page-link">...</span>
      </li>

      <!-- Page numbers -->
      <li class="page-item" v-for="page in displayedPages" :key="page"
          :class="{ active: currentPage === page }">
        <a class="page-link" href="#" @click.prevent="onPageChange(page)">{{ page }}</a>
      </li>

      <!-- Ellipsis if needed -->
      <li class="page-item disabled" v-if="endPage < totalPages - 1">
        <span class="page-link">...</span>
      </li>

      <!-- Last page -->
      <li class="page-item" v-if="totalPages > 1" :class="{ active: currentPage === totalPages }">
        <a class="page-link" href="#" @click.prevent="onPageChange(totalPages)">{{ totalPages }}</a>
      </li>

      <!-- Next button -->
      <li class="page-item" :class="{ disabled: currentPage === totalPages }">
        <a class="page-link" href="#" @click.prevent="onPageChange(currentPage + 1)">
          <span aria-hidden="true">&raquo;</span>
        </a>
      </li>
    </ul>
  </nav>
</template>

<script>
export default {
  name: 'Pagination',
  props: {
    currentPage: {
      type: Number,
      required: true
    },
    totalItems: {
      type: Number,
      required: true
    },
    pageSize: {
      type: Number,
      default: 10
    },
    maxDisplayPages: {
      type: Number,
      default: 5
    }
  },
  computed: {
    totalPages() {
      return Math.ceil(this.totalItems / this.pageSize);
    },
    startPage() {
      // Calculate start page based on current page and max pages to display
      const halfWay = Math.floor(this.maxDisplayPages / 2);
      const isStartValid = this.currentPage - halfWay > 0;
      const isEndValid = this.currentPage + halfWay <= this.totalPages;

      if (!isStartValid) {
        return 2;
      } else if (!isEndValid) {
        return Math.max(2, this.totalPages - this.maxDisplayPages + 1);
      }
      return this.currentPage - halfWay;
    },
    endPage() {
      return Math.min(this.startPage + this.maxDisplayPages - 1, this.totalPages - 1);
    },
    displayedPages() {
      const pages = [];
      for (let i = this.startPage; i <= this.endPage; i++) {
        pages.push(i);
      }
      return pages;
    }
  },
  methods: {
    onPageChange(page) {
      if (page < 1 || page > this.totalPages || page === this.currentPage) {
        return;
      }
      this.$emit('page-changed', page);
    }
  }
}
</script>

<style scoped>
.pagination {
  margin-top: 20px;
  margin-bottom: 20px;
}
</style>