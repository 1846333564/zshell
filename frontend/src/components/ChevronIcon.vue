<template>
  <svg
    class="chevron-icon"
    :class="`chevron-${direction}`"
    viewBox="0 0 12 12"
    fill="none"
    aria-hidden="true"
    focusable="false"
  >
    <path d="M3 1.5 7.5 6 3 10.5" />
  </svg>
</template>

<script setup>
defineProps({
  direction: {
    type: String,
    default: 'right',
    validator: (value) => ['right', 'down', 'left', 'up'].includes(value),
  },
});
</script>
