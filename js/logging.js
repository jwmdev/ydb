
let date = new Date().getTime()

const writeDate = () => {
  const oldDate = date
  date = new Date().getTime()
  return date - oldDate
}

export const print = (...args) => console.log(...args)
export const log = m => print(`%cydb-client: %cdtrn %c${writeDate()}ms`, 'color: blue;', '', 'color: blue')

export const fail = m => {
  throw new Error(m)
}
