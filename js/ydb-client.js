/* eslint-env browser */
import * as idbactions from './idbactions.js'
import * as globals from './globals.js'
import * as message from './message.js'
import * as bc from './broadcastchannel.js'
import * as decoding from './decoding.js'
import * as logging from './logging.js'

const dbPromise = idbactions.openDB()

export class YdbClient {
  constructor (url, db) {
    const ws = new WebSocket(url)
    this.ws = ws
    this.rooms = new Map()
    this.db = db
    ws.onmessage = event => {
      const t = idbactions.createTransaction(db)
      const decoder = decoding.createDecoder(event.data)
      while (decoding.hasContent(decoder)) {
        switch (decoding.readVarUint(decoder)) {
          case message.MESSAGE_UPDATE: {
            const room = decoding.readVarString(decoder)
            const offset = decoding.readVarUint(decoder)
            const update = decoding.readPayload(decoder)
            idbactions.writeHostUnconfirmed(t, room, offset, update)
            bc.publish(room, update)
            break
          }
          case message.MESSAGE_SUB_CONF: {
            const room = decoding.readVarString(decoder)
            const offset = decoding.readVarUint(decoder)
            const roomsid = decoding.readVarUint(decoder)
            const data = decoding.readPayload(decoder)
            idbactions.confirmSubscription(t, room, roomsid, offset, data)
            break
          }
          case message.MESSAGE_CONFIRMATION: {
            const room = decoding.readVarString(decoder)
            const offset = decoding.readVarUint(decoder)
            idbactions.writeConfirmedByHost(t, room, offset)
            break
          }
          default:
            logging.fail(`Unexpected message type`)
        }
      }
    }
  }
}

/**
 * @param {YdbClient} ydb
 * @param {ArrayBuffer} m
 */
const readMessage = (ydb, m) => 

export const getYdb = url => dbPromise.then(db => globals.presolve(new YdbClient(url, db)))

/**
 * @param {YdbClient} ydb
 * @param {ArrayBuffer} m
 */
export const send = (ydb, m) => ydb.ws.send(m)

/**
 * @param {YdbClient} ydb
 * @param {string} room
 * @param {ArrayBuffer} update
 */
export const update = (ydb, room, update) => {
  bc.publish(room, update)
  const t = idbactions.createTransaction(ydb.db)
  return idbactions.writeClientUnconfirmed(t, room, update).then(clientConf =>
    send(ydb, message.createUpdate(room, update, clientConf))
  )
}

export const subscribe = (ydb, room, f) => {
  bc.subscribe(room, f)
  const t = idbactions.createTransaction(ydb.db)
  idbactions.getRoomData(t, room).then(data => {
    if (data.byteLength > 0) {
      f(data)
    }
  })
  idbactions.getRoomMeta(t, room).then(meta => {
    if (meta === undefined) {
      // TODO: maybe set prelim meta value so we don't sub twice
      send(ydb, message.createSub(ydb, [room]))
    }
  })
}
