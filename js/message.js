import * as encoding from './encoding.js'

export const MESSAGE_UPDATE = 0
export const MESSAGE_SUB = 1
export const MESSAGE_CONFIRMATION = 2

/**
 * @param {string} room
 * @param {ArrayBuffer} update
 * @param {number} clientConf
 * @return {ArrayBuffer}
 */
export const createUpdate = (room, update, clientConf) => {
  const encoder = encoding.createEncoder()
  encoding.writeVarUint(encoder, MESSAGE_UPDATE)
  encoding.writeVarUint(encoder, clientConf)
  encoding.writeVarString(encoder, room)
  encoding.writePayload(encoder, update)
  return encoding.toBuffer(encoder)
}

/**
 * @typedef SubDef
 * @type {Object}
 * @property {string} room
 * @property {number} offset
 */

/**
 * @param {Array<SubDef>} rooms
 * @return {ArrayBuffer}
 */
export const createSub = rooms => {
  const encoder = encoding.createEncoder()
  encoding.writeVarUint(encoder, MESSAGE_SUB)
  encoding.writeVarUint(encoder, rooms.length)
  for (let i = 0; i < rooms.length; i++) {
    encoding.writeVarString(encoder, rooms[i].room)
    encoding.writeVarUint(encoder, rooms[i].offset)
  }
  return encoding.toBuffer(encoder)
}
