import * as logging from './logging.js'

export const run = (name, f) => {
  console.info(name)
  const start = new Date()
  f(name)
  logging.print(`%cSuccess:%c ${name} in %c${new Date().getTime() - start.getTime()}ms`, 'color:green;font-weight:bold', '', 'color:grey')
}

export const compareArrays = (as, bs) => {
  if (as.length !== bs.length) {
    return false
  }
  for (let i = 0; i < as.length; i++) {
    if (as[i] !== bs[i]) {
      return false
    }
  }
  return true
}
