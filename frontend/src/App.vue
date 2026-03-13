<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useNoticeStore } from './stores/notice'

const noticeStore = useNoticeStore()
const { current, visible } = storeToRefs(noticeStore)

const noticeClass = computed(() => {
  if (!current.value) return ''
  if (current.value.type === 'success') return 'is-success'
  if (current.value.type === 'error') return 'is-error'
  return 'is-info'
})
</script>

<template>
  <div class="app-root">
    <div class="app-top-notice-host">
      <transition name="app-top-notice-slide">
        <div v-if="current && visible" :key="current.id" class="app-top-notice" :class="noticeClass">
          {{ current.text }}
        </div>
      </transition>
    </div>
    <router-view />
  </div>
</template>
