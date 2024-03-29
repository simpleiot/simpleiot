// source: serial.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global =
    (typeof globalThis !== 'undefined' && globalThis) ||
    (typeof window !== 'undefined' && window) ||
    (typeof global !== 'undefined' && global) ||
    (typeof self !== 'undefined' && self) ||
    (function () { return this; }).call(null) ||
    Function('return this')();

var point_pb = require('./point_pb.js');
goog.object.extend(proto, point_pb);
goog.exportSymbol('proto.pb.Serial', null, global);
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.pb.Serial = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.pb.Serial.repeatedFields_, null);
};
goog.inherits(proto.pb.Serial, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.pb.Serial.displayName = 'proto.pb.Serial';
}

/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.pb.Serial.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.pb.Serial.prototype.toObject = function(opt_includeInstance) {
  return proto.pb.Serial.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.pb.Serial} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.pb.Serial.toObject = function(includeInstance, msg) {
  var f, obj = {
    subject: jspb.Message.getFieldWithDefault(msg, 1, ""),
    pointsList: jspb.Message.toObjectList(msg.getPointsList(),
    point_pb.SerialPoint.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.pb.Serial}
 */
proto.pb.Serial.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.pb.Serial;
  return proto.pb.Serial.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.pb.Serial} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.pb.Serial}
 */
proto.pb.Serial.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setSubject(value);
      break;
    case 2:
      var value = new point_pb.SerialPoint;
      reader.readMessage(value,point_pb.SerialPoint.deserializeBinaryFromReader);
      msg.addPoints(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.pb.Serial.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.pb.Serial.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.pb.Serial} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.pb.Serial.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSubject();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getPointsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      2,
      f,
      point_pb.SerialPoint.serializeBinaryToWriter
    );
  }
};


/**
 * optional string subject = 1;
 * @return {string}
 */
proto.pb.Serial.prototype.getSubject = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.pb.Serial} returns this
 */
proto.pb.Serial.prototype.setSubject = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated SerialPoint points = 2;
 * @return {!Array<!proto.pb.SerialPoint>}
 */
proto.pb.Serial.prototype.getPointsList = function() {
  return /** @type{!Array<!proto.pb.SerialPoint>} */ (
    jspb.Message.getRepeatedWrapperField(this, point_pb.SerialPoint, 2));
};


/**
 * @param {!Array<!proto.pb.SerialPoint>} value
 * @return {!proto.pb.Serial} returns this
*/
proto.pb.Serial.prototype.setPointsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 2, value);
};


/**
 * @param {!proto.pb.SerialPoint=} opt_value
 * @param {number=} opt_index
 * @return {!proto.pb.SerialPoint}
 */
proto.pb.Serial.prototype.addPoints = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 2, opt_value, proto.pb.SerialPoint, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.pb.Serial} returns this
 */
proto.pb.Serial.prototype.clearPointsList = function() {
  return this.setPointsList([]);
};


goog.object.extend(exports, proto.pb);
