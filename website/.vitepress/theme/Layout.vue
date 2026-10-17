<script setup lang="ts">
import DefaultTheme from 'vitepress/theme'
import AiNotice from './components/AiNotice.vue'

const { Layout } = DefaultTheme
</script>

<template>
  <Layout>
    <template #doc-before>
      <AiNotice />
    </template>
  </Layout>
</template>
