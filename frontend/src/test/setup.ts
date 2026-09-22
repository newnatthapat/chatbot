import '@testing-library/jest-dom/vitest'

// jsdom doesn't implement scrollIntoView; ChatPage calls it to auto-scroll
// the message list, so tests need a no-op stand-in.
Element.prototype.scrollIntoView ??= function scrollIntoView() {}
