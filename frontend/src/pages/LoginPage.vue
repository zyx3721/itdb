<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useNoticeStore } from '../stores/notice'

const router = useRouter()
const auth = useAuthStore()
const noticeStore = useNoticeStore()

const USERNAME_CACHE_KEY = 'itdb_login_username'
const REMEMBER_FLAG_KEY = 'itdb_login_remember'

const username = ref('')
const password = ref('')
const rememberMe = ref(false)
const loading = ref(false)
const showPassword = ref(false)
const loginMode = ref<'local' | 'ldap'>('local')
const activeField = ref<'username' | 'password' | null>(null)
const usernamePhase = ref<'glance' | 'focus' | null>(null)
let usernamePhaseTimerId = 0

const mouseX = ref(0)
const mouseY = ref(0)
const isPurpleBlinking = ref(false)
const isBlackBlinking = ref(false)
const isPurplePeeking = ref(false)

const purpleRef = ref<HTMLElement | null>(null)
const blackRef = ref<HTMLElement | null>(null)
const orangeRef = ref<HTMLElement | null>(null)
const yellowRef = ref<HTMLElement | null>(null)

const timerIds = new Set<number>()
let purplePeekToken = 0

const isUsernameActive = computed(() => activeField.value === 'username')
const isPasswordActive = computed(() => activeField.value === 'password')
const isRevealMode = computed(() => showPassword.value && password.value.trim().length > 0)
const isConcealMode = computed(() => isPasswordActive.value && !showPassword.value)

function trackTimeout(fn: () => void, delay: number) {
  const id = window.setTimeout(() => {
    timerIds.delete(id)
    fn()
  }, delay)
  timerIds.add(id)
  return id
}

function clearAllTimers() {
  for (const id of timerIds) {
    window.clearTimeout(id)
  }
  timerIds.clear()
}

function startBlinkLoop(target: typeof isPurpleBlinking) {
  const schedule = () => {
    trackTimeout(() => {
      target.value = true
      trackTimeout(() => {
        target.value = false
        schedule()
      }, 160)
    }, 3000 + Math.random() * 4000)
  }
  schedule()
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

function getCharacterPose(el: HTMLElement | null) {
  if (!el) return { faceX: 0, faceY: 0, bodySkew: 0 }

  const rect = el.getBoundingClientRect()
  const centerX = rect.left + rect.width / 2
  const centerY = rect.top + rect.height / 3
  const deltaX = mouseX.value - centerX
  const deltaY = mouseY.value - centerY

  return {
    faceX: clamp(deltaX / 20, -15, 15),
    faceY: clamp(deltaY / 30, -10, 10),
    bodySkew: clamp(-deltaX / 120, -6, 6),
  }
}

const purplePose = computed(() => getCharacterPose(purpleRef.value))
const blackPose = computed(() => getCharacterPose(blackRef.value))
const orangePose = computed(() => getCharacterPose(orangeRef.value))
const yellowPose = computed(() => getCharacterPose(yellowRef.value))

const purpleEyeLook = computed(() => {
  if (isRevealMode.value) {
    return isPurplePeeking.value ? { x: 4, y: 5 } : { x: -4, y: -4 }
  }
  if (isUsernameActive.value) {
    if (usernamePhase.value === 'glance') {
      return { x: 5, y: 3 }
    }
    // focus 阶段跟随鼠标，与 concealMode 一致
    const pose = purplePose.value
    return {
      x: clamp(pose.faceX * 0.3, -4, 4),
      y: clamp(pose.faceY * 0.4, -3.5, 3.5),
    }
  }
  const pose = purplePose.value
  return {
    x: clamp(pose.faceX * 0.3, -4, 4),
    y: clamp(pose.faceY * 0.4, -3.5, 3.5),
  }
})

const blackEyeLook = computed(() => {
  if (isRevealMode.value) return { x: -4, y: -4 }
  if (isUsernameActive.value) {
    if (usernamePhase.value === 'glance') {
      return { x: -4.5, y: -3 }
    }
    // focus 阶段跟随鼠标，与 concealMode 一致
    const pose = blackPose.value
    return {
      x: clamp(pose.faceX * 0.28, -3.5, 3.5),
      y: clamp(pose.faceY * 0.35, -3, 3),
    }
  }
  const pose = blackPose.value
  return {
    x: clamp(pose.faceX * 0.28, -3.5, 3.5),
    y: clamp(pose.faceY * 0.35, -3, 3),
  }
})

const orangeEyeLook = computed(() => {
  if (isRevealMode.value) return { x: -5, y: -4 }
  return null
})

const yellowEyeLook = computed(() => {
  if (isRevealMode.value) return { x: -5, y: -4 }
  return null
})

function setActiveField(field: 'username' | 'password' | null) {
  activeField.value = field

  if (usernamePhaseTimerId) {
    window.clearTimeout(usernamePhaseTimerId)
    timerIds.delete(usernamePhaseTimerId)
    usernamePhaseTimerId = 0
  }

  if (field === 'username') {
    usernamePhase.value = 'glance'
    usernamePhaseTimerId = trackTimeout(() => {
      usernamePhase.value = 'focus'
      usernamePhaseTimerId = 0
    }, 1300)
  } else {
    usernamePhase.value = null
  }
}

function handleMouseMove(event: MouseEvent) {
  mouseX.value = event.clientX
  mouseY.value = event.clientY
}

onMounted(() => {
  const remembered = localStorage.getItem(REMEMBER_FLAG_KEY) === '1'
  rememberMe.value = remembered
  if (remembered) {
    username.value = localStorage.getItem(USERNAME_CACHE_KEY) || ''
  }

  mouseX.value = window.innerWidth / 2
  mouseY.value = window.innerHeight / 2

  window.addEventListener('mousemove', handleMouseMove)
  startBlinkLoop(isPurpleBlinking)
  startBlinkLoop(isBlackBlinking)
})

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', handleMouseMove)
  clearAllTimers()
})

watch(rememberMe, (checked) => {
  if (!checked) {
    localStorage.removeItem(REMEMBER_FLAG_KEY)
    localStorage.removeItem(USERNAME_CACHE_KEY)
  }
})

watch([password, showPassword], ([nextPassword, nextShowPassword]) => {
  purplePeekToken += 1
  const token = purplePeekToken
  isPurplePeeking.value = false

  if (!nextPassword.trim() || !nextShowPassword) return

  const schedulePeek = () => {
    trackTimeout(() => {
      if (purplePeekToken !== token) return
      isPurplePeeking.value = true
      trackTimeout(() => {
        if (purplePeekToken !== token) return
        isPurplePeeking.value = false
        schedulePeek()
      }, 820)
    }, 2200 + Math.random() * 2800)
  }

  schedulePeek()
})

async function onLogin() {
  const loginUsername = username.value.trim()
  if (!loginUsername || !password.value.trim()) {
    noticeStore.error('请完整填写登录信息')
    return
  }

  await auth.login(loginUsername, password.value, loginMode.value)
  await auth.loadMe()

  if (rememberMe.value) {
    localStorage.setItem(REMEMBER_FLAG_KEY, '1')
    localStorage.setItem(USERNAME_CACHE_KEY, loginUsername)
  } else {
    localStorage.removeItem(REMEMBER_FLAG_KEY)
    localStorage.removeItem(USERNAME_CACHE_KEY)
  }

  noticeStore.success('登录成功')
  await new Promise((resolve) => setTimeout(resolve, 220))
  await router.replace('/dashboard')
}

async function onPrimaryAction() {
  loading.value = true
  try {
    await onLogin()
  } catch (err: unknown) {
    if (!(err as { response?: unknown })?.response) {
      const message = (err as { message?: string })?.message ?? '登录失败'
      noticeStore.error(message)
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-wrap" @keyup.enter="onPrimaryAction">
    <div class="login-layout">
      <section class="login-scene-panel" aria-hidden="true">
        <div class="scene-brand">
          <img class="scene-brand-logo" src="/images/favicon.png" alt="ITDB" />
          <strong>ITDB</strong>
        </div>

        <div class="scene-stage-wrap">
          <div class="character-stage">
          <div class="stage-glow stage-glow-one"></div>
          <div class="stage-glow stage-glow-two"></div>

          <div
            ref="purpleRef"
            class="character character-purple"
            :style="{
              height: isConcealMode || (isUsernameActive && usernamePhase === 'focus') ? '440px'
                : isUsernameActive && usernamePhase === 'glance' ? '430px'
                : '400px',
              transform: isRevealMode
                ? 'skewX(0deg)'
                : isConcealMode || (isUsernameActive && usernamePhase === 'focus')
                  ? `skewX(${purplePose.bodySkew - 12}deg) translateX(40px)`
                  : isUsernameActive && usernamePhase === 'glance'
                    ? `skewX(${purplePose.bodySkew - 3}deg) translateX(15px)`
                    : `skewX(${purplePose.bodySkew}deg)`,
            }"
          >
            <div
              class="character-eyes"
              :style="{
                left: isRevealMode ? '20px'
                  : isUsernameActive && usernamePhase === 'glance' ? '60px'
                  : `${45 + purplePose.faceX}px`,
                top: isRevealMode ? '35px'
                  : isUsernameActive && usernamePhase === 'glance' ? '42px'
                  : `${40 + purplePose.faceY}px`,
              }"
            >
              <div class="eye eye-shell" :class="{ blink: isPurpleBlinking }">
                <div
                  v-if="!isPurpleBlinking"
                  class="eye-pupil eye-pupil-shell"
                  :style="purpleEyeLook ? { transform: `translate(${purpleEyeLook.x}px, ${purpleEyeLook.y}px)` } : undefined"
                ></div>
              </div>
              <div class="eye eye-shell" :class="{ blink: isPurpleBlinking }">
                <div
                  v-if="!isPurpleBlinking"
                  class="eye-pupil eye-pupil-shell"
                  :style="purpleEyeLook ? { transform: `translate(${purpleEyeLook.x}px, ${purpleEyeLook.y}px)` } : undefined"
                ></div>
              </div>
            </div>
          </div>

          <div
            ref="blackRef"
            class="character character-black"
            :style="{
              transform: isRevealMode
                ? 'skewX(0deg)'
                : isUsernameActive && usernamePhase === 'glance'
                  ? `skewX(${blackPose.bodySkew - 4}deg)`
                  : isUsernameActive && usernamePhase === 'focus'
                    ? `skewX(${blackPose.bodySkew * 1.5}deg)`
                    : isConcealMode
                      ? `skewX(${blackPose.bodySkew * 1.5}deg)`
                      : `skewX(${blackPose.bodySkew}deg)`,
            }"
          >
            <div
              class="character-eyes character-eyes-compact"
              :style="{
                left: isRevealMode ? '10px'
                  : isUsernameActive && usernamePhase === 'glance' ? '18px'
                  : `${26 + blackPose.faceX}px`,
                top: isRevealMode ? '28px'
                  : isUsernameActive && usernamePhase === 'glance' ? '32px'
                  : `${32 + blackPose.faceY}px`,
              }"
            >
              <div class="eye eye-shell eye-small" :class="{ blink: isBlackBlinking }">
                <div
                  v-if="!isBlackBlinking"
                  class="eye-pupil eye-pupil-shell eye-pupil-small"
                  :style="blackEyeLook ? { transform: `translate(${blackEyeLook.x}px, ${blackEyeLook.y}px)` } : undefined"
                ></div>
              </div>
              <div class="eye eye-shell eye-small" :class="{ blink: isBlackBlinking }">
                <div
                  v-if="!isBlackBlinking"
                  class="eye-pupil eye-pupil-shell eye-pupil-small"
                  :style="blackEyeLook ? { transform: `translate(${blackEyeLook.x}px, ${blackEyeLook.y}px)` } : undefined"
                ></div>
              </div>
            </div>
          </div>

          <div
            ref="orangeRef"
            class="character character-orange"
            :style="{ transform: isRevealMode ? 'skewX(0deg)' : `skewX(${orangePose.bodySkew}deg)` }"
          >
            <div
              class="character-eyes character-eyes-pupils character-eyes-pupils-wide"
              :style="{
                left: isRevealMode ? '50px' : `${82 + orangePose.faceX}px`,
                top: isRevealMode ? '85px' : `${90 + orangePose.faceY}px`,
              }"
            >
              <div class="eye-pupil eye-pupil-plain" :style="orangeEyeLook ? { transform: `translate(${orangeEyeLook.x}px, ${orangeEyeLook.y}px)` } : undefined"></div>
              <div class="eye-pupil eye-pupil-plain" :style="orangeEyeLook ? { transform: `translate(${orangeEyeLook.x}px, ${orangeEyeLook.y}px)` } : undefined"></div>
            </div>
          </div>

          <div
            ref="yellowRef"
            class="character character-yellow"
            :style="{ transform: isRevealMode ? 'skewX(0deg)' : `skewX(${yellowPose.bodySkew}deg)` }"
          >
            <div
              class="character-eyes character-eyes-pupils"
              :style="{
                left: isRevealMode ? '20px' : `${52 + yellowPose.faceX}px`,
                top: isRevealMode ? '35px' : `${40 + yellowPose.faceY}px`,
              }"
            >
              <div class="eye-pupil eye-pupil-plain" :style="yellowEyeLook ? { transform: `translate(${yellowEyeLook.x}px, ${yellowEyeLook.y}px)` } : undefined"></div>
              <div class="eye-pupil eye-pupil-plain" :style="yellowEyeLook ? { transform: `translate(${yellowEyeLook.x}px, ${yellowEyeLook.y}px)` } : undefined"></div>
            </div>
            <div
              class="character-mouth"
              :style="{
                left: isRevealMode ? '10px' : `${40 + yellowPose.faceX}px`,
                top: isRevealMode ? '88px' : `${88 + yellowPose.faceY}px`,
              }"
            ></div>
          </div>
          </div>
        </div>
      </section>

      <section class="login-panel">
        <transition name="auth-panel" appear>
          <div class="auth-panel-wrap">
            <div class="login-card">
              <header class="login-header">
                <div class="login-mode-switch" role="tablist" aria-label="登录方式">
                  <button
                    class="login-mode-btn"
                    :class="{ active: loginMode === 'local' }"
                    type="button"
                    @click="loginMode = 'local'"
                  >
                    普通登录
                  </button>
                  <button
                    class="login-mode-btn"
                    :class="{ active: loginMode === 'ldap' }"
                    type="button"
                    @click="loginMode = 'ldap'"
                  >
                    LDAP 登录
                  </button>
                </div>
                <p>欢迎使用 ITDB 资产管理系统</p>
              </header>

              <form class="login-form" @submit.prevent="onPrimaryAction">
                <div class="field-row">
                  <label class="field-label" for="login-username">用户名</label>
                  <input
                    id="login-username"
                    v-model="username"
                    type="text"
                    autocomplete="username"
                    placeholder="请输入用户名"
                    @focus="setActiveField('username')"
                    @blur="setActiveField(null)"
                  />
                </div>

                <div class="field-row">
                  <label class="field-label" for="login-password">密码</label>
                  <div class="field-input-wrap">
                    <input
                      id="login-password"
                      v-model="password"
                      :type="showPassword ? 'text' : 'password'"
                      autocomplete="current-password"
                      placeholder="请输入密码"
                      @focus="setActiveField('password')"
                      @blur="setActiveField(null)"
                    />
                    <button
                      class="field-visibility-btn"
                      type="button"
                      :aria-label="showPassword ? '隐藏密码' : '显示密码'"
                      @click="showPassword = !showPassword"
                    >
                      <svg v-if="!showPassword" viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M1.5 12s3.9-6.5 10.5-6.5S22.5 12 22.5 12 18.6 18.5 12 18.5 1.5 12 1.5 12Z" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
                        <circle cx="12" cy="12" r="3.2" fill="none" stroke="currentColor" stroke-width="1.8"/>
                      </svg>
                      <svg v-else viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M3 3l18 18" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
                        <path d="M10.7 5.8A11.2 11.2 0 0 1 12 5.5c6.6 0 10.5 6.5 10.5 6.5a18 18 0 0 1-3.6 4.3" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
                        <path d="M6.1 6.2C3.3 8 1.5 12 1.5 12s3.9 6.5 10.5 6.5c1.7 0 3.1-.4 4.4-1" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
                        <path d="M9.9 9.9A3.2 3.2 0 0 0 14.1 14.1" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
                      </svg>
                    </button>
                  </div>
                </div>

                <div class="login-options">
                  <label class="remember-box" for="remember-account">
                    <input id="remember-account" v-model="rememberMe" type="checkbox" />
                    <span>记住账号</span>
                  </label>
                </div>

                <button class="login-submit" :disabled="loading" type="submit">
                  {{ loading ? '登录中...' : '登录' }}
                </button>
              </form>
            </div>
          </div>
        </transition>
      </section>
    </div>
  </div>
</template>

<style scoped>
.login-wrap {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
  background:
    radial-gradient(920px 480px at 2% 4%, rgba(39, 101, 173, 0.18), transparent 62%),
    radial-gradient(760px 360px at 98% 2%, rgba(20, 164, 141, 0.12), transparent 58%),
    linear-gradient(145deg, #edf6ff 0%, #f6fbff 48%, #eef8f6 100%);
}

.login-layout {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 1fr 1fr;
}

.login-scene-panel {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 32px 36px 22px;
  background: linear-gradient(160deg, #173f68 0%, #234f82 38%, #27588d 100%);
  color: rgba(255, 255, 255, 0.96);
}

.login-scene-panel::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image: radial-gradient(rgba(255, 255, 255, 0.09) 1px, transparent 1px);
  background-size: 20px 20px;
  opacity: 0.5;
  pointer-events: none;
}

.scene-brand,
.scene-stage-wrap,
.character-stage {
  position: relative;
  z-index: 1;
}

.scene-brand {
  display: inline-flex;
  align-items: center;
  gap: 12px;
}

.scene-brand-logo {
  width: 40px;
  height: 40px;
  object-fit: contain;
  border-radius: 12px;
  padding: 5px;
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.scene-brand strong {
  display: block;
  font-size: 16px;
  letter-spacing: 0.08em;
}

.scene-stage-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.character-stage {
  width: min(100%, 550px);
  height: 400px;
  margin-top: 0;
  transform: translateY(-18px);
}

.stage-glow {
  position: absolute;
  border-radius: 999px;
  pointer-events: none;
  filter: blur(22px);
}

.stage-glow-one {
  width: 240px;
  height: 240px;
  right: 30px;
  top: 10px;
  background: rgba(255, 255, 255, 0.14);
}

.stage-glow-two {
  width: 320px;
  height: 320px;
  left: -10px;
  bottom: -30px;
  background: rgba(115, 176, 232, 0.2);
}

.character {
  position: absolute;
  bottom: 0;
  transition: transform 0.7s ease, height 0.7s ease;
  transform-origin: bottom center;
}

.character-purple {
  left: 72px;
  width: 180px;
  height: 400px;
  border-radius: 12px 12px 0 0;
  background: #6c3ff5;
  z-index: 1;
}

.character-black {
  left: 244px;
  width: 122px;
  height: 310px;
  border-radius: 8px 8px 0 0;
  background: #2d2d2d;
  z-index: 2;
}

.character-orange {
  left: 0;
  width: 240px;
  height: 200px;
  border-radius: 120px 120px 0 0;
  background: #ff9b6b;
  z-index: 3;
}

.character-yellow {
  left: 316px;
  width: 140px;
  height: 230px;
  border-radius: 70px 70px 0 0;
  background: #e8d754;
  z-index: 4;
}

.character-eyes {
  position: absolute;
  display: flex;
  gap: 32px;
  transition: left 0.2s ease, top 0.2s ease;
}

.character-eyes-compact {
  gap: 24px;
}

.character-eyes-pupils {
  gap: 24px;
}

.character-eyes-pupils-wide {
  gap: 32px;
}

.eye {
  width: 18px;
  height: 18px;
  border-radius: 999px;
  display: grid;
  place-items: center;
  overflow: hidden;
  transition: height 0.16s ease;
}

.eye-shell {
  background: #ffffff;
}

.eye-small {
  width: 16px;
  height: 16px;
}

.eye.blink {
  height: 2px;
}

.eye-pupil {
  border-radius: 999px;
  transition: transform 0.1s ease-out;
}

.eye-pupil-shell {
  width: 7px;
  height: 7px;
  background: #2d2d2d;
}

.eye-pupil-small {
  width: 6px;
  height: 6px;
}

.eye-pupil-plain {
  width: 12px;
  height: 12px;
  background: #2d2d2d;
}

.character-mouth {
  position: absolute;
  width: 80px;
  height: 4px;
  border-radius: 999px;
  background: #2d2d2d;
  transition: left 0.2s ease, top 0.2s ease;
}

.login-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 28px 34px;
}

.auth-panel-wrap {
  width: 100%;
  max-width: 560px;
}

.login-card {
  position: relative;
  z-index: 1;
  width: 100%;
  border-radius: 26px;
  background: rgba(255, 255, 255, 0.92);
  border: 1px solid rgba(207, 225, 239, 0.92);
  box-shadow: 0 28px 62px rgba(13, 55, 90, 0.14);
  backdrop-filter: blur(6px);
  padding: 28px 56px 28px;
}

.login-header {
  display: grid;
  justify-items: center;
}

.login-mode-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px;
  border-radius: 999px;
  margin: 0 auto;
  background: rgba(227, 238, 248, 0.86);
  border: 1px solid rgba(190, 208, 224, 0.96);
}

.login-mode-btn {
  min-width: 108px;
  height: 36px;
  border-radius: 999px;
  border: 1px solid transparent;
  background: transparent;
  color: #48647c;
  font-size: 15px;
  padding: 0 14px;
  font-weight: 600;
  box-shadow: none;
}

.login-mode-btn.active {
  color: #103553;
  border-color: rgba(30, 107, 168, 0.24);
  background: linear-gradient(180deg, #ffffff, #eef6fc);
  box-shadow: 0 6px 14px rgba(15, 53, 84, 0.08);
}

.login-header p {
  color: #55748f;
  margin: 14px 0 20px;
  font-size: 16px;
  text-align: center;
}

.login-form {
  width: 100%;
  max-width: 500px;
  margin: 0 auto;
  display: grid;
  gap: 0;
}

.field-row {
  display: grid;
  width: 100%;
  grid-template-columns: 92px minmax(0, 1fr);
  align-items: center;
  column-gap: 12px;
}

.field-row + .field-row {
  margin-top: 16px;
}

.field-label {
  width: 92px;
  text-align: right;
  color: #214566;
  font-weight: 600;
  font-size: 16px;
  line-height: 1.25;
}

.field-input-wrap {
  position: relative;
}

.field-row input {
  width: 100%;
  height: 46px;
  border-radius: 12px;
  border: 1px solid #cad7e4;
  background: #ffffff;
  font-size: 16px;
  padding: 0 14px;
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.field-input-wrap input {
  padding-right: 46px;
}

.field-row input:focus {
  outline: none;
  border-color: #2f7fba;
  box-shadow: 0 0 0 2px rgba(47, 127, 186, 0.15);
}

.field-visibility-btn {
  position: absolute;
  top: 50%;
  right: 25px;
  width: 28px;
  height: 28px;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: #5d7990;
  transform: translateY(-50%);
}

.field-visibility-btn:hover {
  color: #244968;
}

.field-visibility-btn svg {
  width: 80px;
  height: 20px;
  display: block;
}

.login-options {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 34px;
  margin: 15px 0 15px;
}

.remember-box {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  white-space: nowrap;
  font-size: 15px;
  color: #405a70;
  font-weight: 500;
  text-align: center;
  cursor: pointer;
}

.remember-box input {
  width: 16px;
  height: 16px;
  margin: 0;
  cursor: pointer;
}

.login-submit {
  margin: 2px auto 0;
  width: 100%;
  height: 46px;
  border: 1px solid #1e6ba8;
  border-radius: 12px;
  background: linear-gradient(180deg, #2d7eb9 0%, #1f6ea7 100%);
  color: #ffffff;
  font-size: 17px;
  font-weight: 600;
}

.login-submit:hover {
  background: linear-gradient(180deg, #2a79b3 0%, #1b6396 100%);
}

.login-submit:disabled {
  opacity: 0.65;
}

.auth-panel-enter-active {
  transition:
    opacity 0.72s cubic-bezier(0.22, 0.61, 0.36, 1),
    transform 0.72s cubic-bezier(0.22, 0.61, 0.36, 1);
}

.auth-panel-enter-from {
  opacity: 0;
  transform: translateY(-28px);
}

.auth-panel-enter-to {
  opacity: 1;
  transform: translateY(0);
}

@media (max-width: 1180px) {
  .login-scene-panel {
    padding-inline: 28px;
  }

  .character-stage {
    transform: scale(0.92);
    transform-origin: center bottom;
  }

  .login-card {
    padding-inline: 36px;
  }
}

@media (max-width: 900px) {
  .login-layout {
    grid-template-columns: 1fr;
  }

  .login-scene-panel {
    display: none;
  }

  .login-panel {
    min-height: 100vh;
    padding: 18px;
  }

  .auth-panel-wrap {
    max-width: 100%;
  }
}

@media (max-width: 768px) {
  .login-card {
    border-radius: 18px;
    padding: 20px 16px 12px;
  }

  .login-form {
    gap: 12px;
  }

  .field-row {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .field-label {
    width: auto;
    text-align: left;
    font-size: 15px;
  }

  .field-row + .field-row {
    margin-top: 0;
  }

  .login-mode-switch {
    width: 100%;
    justify-content: stretch;
  }

  .login-mode-btn {
    flex: 1 1 0;
    min-width: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .character,
  .character-eyes,
  .character-mouth,
  .eye,
  .eye-pupil,
  .auth-panel-enter-active {
    transition: none !important;
  }
}
</style>
