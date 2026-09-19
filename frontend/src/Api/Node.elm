module Api.Node exposing
    ( EdgeRole(..)
    , GeneratedToken
    , Node
    , NodeView
    , Notification
    , copy
    , decode
    , delete
    , description
    , edgeRole
    , generateKey
    , getBestDesc
    , insert
    , move
    , notify
    , owningParentType
    , typeAction
    , typeActionInactive
    , typeBrowser
    , typeCanBus
    , typeCondition
    , typeDb
    , typeDevice
    , typeDeviceCred
    , typeEnrollToken
    , typeFile
    , typeGpio
    , typeGps
    , typeGroup
    , typeIio
    , typeIioChannel
    , typeMetrics
    , typeModbus
    , typeModbusIO
    , typeMqtt
    , typeMqttDevice
    , typeMqttSub
    , typeMsgService
    , typeNTP
    , typeNetworkManager
    , typeNetworkManagerConn
    , typeNetworkManagerDevice
    , typeOneWire
    , typeParticle
    , typeProvisioning
    , typeProvisioningFile
    , typeRule
    , typeSerialDev
    , typeShelly
    , typeShellyIO
    , typeSignalGenerator
    , typeSparkplugDevice
    , typeSparkplugGroup
    , typeSparkplugNode
    , typeSync
    , typeUpdate
    , typeUser
    , typeVariable
    )

import Api.Data exposing (Data)
import Api.Point as Point exposing (Point)
import Api.Response as Response exposing (Response)
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)
import Json.Encode as Encode
import Url.Builder


typeDevice : String
typeDevice =
    "device"


typeDeviceCred : String
typeDeviceCred =
    "deviceCred"


typeEnrollToken : String
typeEnrollToken =
    "enrollToken"


typeGroup : String
typeGroup =
    "group"


typeRule : String
typeRule =
    "rule"


typeCondition : String
typeCondition =
    "condition"


typeAction : String
typeAction =
    "action"


typeActionInactive : String
typeActionInactive =
    "actionInactive"


typeUser : String
typeUser =
    "user"


typeMsgService : String
typeMsgService =
    "msgService"


typeDb : String
typeDb =
    "db"


typeParticle : String
typeParticle =
    "particle"


typeShelly : String
typeShelly =
    "shelly"


typeShellyIO : String
typeShellyIO =
    "shellyIo"


typeModbus : String
typeModbus =
    "modbus"


typeModbusIO : String
typeModbusIO =
    "modbusIo"


typeSparkplugGroup : String
typeSparkplugGroup =
    "sparkplugGroup"


typeSparkplugNode : String
typeSparkplugNode =
    "sparkplugNode"


typeSparkplugDevice : String
typeSparkplugDevice =
    "sparkplugDevice"


typeMqtt : String
typeMqtt =
    "mqtt"


typeMqttSub : String
typeMqttSub =
    "mqttSub"


typeMqttDevice : String
typeMqttDevice =
    "mqttDevice"


typeIio : String
typeIio =
    "iio"


typeIioChannel : String
typeIioChannel =
    "iioChannel"


typeOneWire : String
typeOneWire =
    "oneWire"


typeOneWireIO : String
typeOneWireIO =
    "oneWireIO"


typeSerialDev : String
typeSerialDev =
    "serialDev"


typeCanBus : String
typeCanBus =
    "canBus"


typeVariable : String
typeVariable =
    "variable"


typeSync : String
typeSync =
    "sync"


typeSignalGenerator : String
typeSignalGenerator =
    "signalGenerator"


typeFile : String
typeFile =
    "file"


typeProvisioning : String
typeProvisioning =
    "provisioning"


typeProvisioningFile : String
typeProvisioningFile =
    "provisioningFile"


typeMetrics : String
typeMetrics =
    "metrics"


typeNetworkManager : String
typeNetworkManager =
    "networkManager"


typeNetworkManagerDevice : String
typeNetworkManagerDevice =
    "networkManagerDevice"


typeNetworkManagerConn : String
typeNetworkManagerConn =
    "networkManagerConn"


typeNTP : String
typeNTP =
    "ntp"


typeGps : String
typeGps =
    "gps"


typeGpio : String
typeGpio =
    "gpio"


typeBrowser : String
typeBrowser =
    "browser"


typeUpdate : String
typeUpdate =
    "update"



-- Node corresponds with Go NodeEdge struct


type alias Node =
    { id : String
    , typ : String
    , hash : Int
    , parent : String
    , points : List Point
    , edgePoints : List Point
    }


{-| What an edge means for the node below it. A node that owns something
outside the tree -- a bus, a line, a socket -- has one primary edge that runs
its client, and any number of mirror edges that display it and run nothing.
A node with no primary location, such as a user, has edges with no role.
-}
type EdgeRole
    = EdgeRoleNone
    | EdgeRolePrimary
    | EdgeRoleMirror


{-| Read the role of the edge a node was reached through. An edge carrying
both points reads as a mirror, matching the backend.
-}
edgeRole : Node -> EdgeRole
edgeRole node =
    if Point.getBool node.edgePoints Point.typeMirror "" then
        EdgeRoleMirror

    else if Point.getBool node.edgePoints Point.typePrimary "" then
        EdgeRolePrimary

    else
        EdgeRoleNone


{-| The parent type a node of this type must live under, or "" when it may
live anywhere. These nodes are found by walking down from their parent rather
than from the tree root, so moving one elsewhere leaves it inert. Mirroring is
what was wanted in that case. Mirrors the nodeTypeOwners table in Go.
-}
owningParentType : String -> String
owningParentType typ =
    if typ == typeModbusIO then
        typeModbus

    else if typ == typeOneWireIO then
        typeOneWire

    else if typ == typeIioChannel then
        typeIio

    else if typ == typeShellyIO then
        typeShelly

    else if typ == typeMqttSub then
        typeMqtt

    else if typ == typeCondition || typ == typeAction || typ == typeActionInactive then
        typeRule

    else if typ == typeNetworkManagerDevice || typ == typeNetworkManagerConn then
        typeNetworkManager

    else if typ == typeProvisioningFile then
        typeProvisioning

    else if typ == typeSparkplugGroup then
        typeMqtt

    else if typ == typeSparkplugNode then
        typeSparkplugGroup

    else if typ == typeSparkplugDevice then
        typeSparkplugNode

    else
        ""


{-| NodeView is a node as the tree shows it. `anchor` is the group the
node was fetched under, which every request about it names. `loading` is
set while its children are being fetched.
-}
type alias NodeView =
    { node : Node
    , feID : Int
    , parentID : String
    , anchor : String
    , hasChildren : Bool
    , expDetail : Bool
    , expChildren : Bool
    , loading : Bool
    , mod : Bool
    }


type alias NodeMove =
    { id : String
    , oldParent : String
    , newParent : String
    }


type alias NodeCopy =
    { id : String
    , oldParent : String
    , newParent : String
    , duplicate : Bool
    }


type alias NodeDelete =
    { parent : String
    }


type alias Notification =
    { id : String
    , sourceNode : String
    , subject : String
    , message : String
    }


decode : Decode.Decoder Node
decode =
    Decode.succeed Node
        |> required "id" Decode.string
        |> required "type" Decode.string
        |> optional "hash" Decode.int 0
        |> required "parent" Decode.string
        |> optional "points" (Decode.list Point.decode) []
        |> optional "edgePoints" (Decode.list Point.decode) []


encode : Node -> Encode.Value
encode node =
    Encode.object
        [ ( "id", Encode.string node.id )
        , ( "type", Encode.string node.typ )
        , ( "hash", Encode.int node.hash )
        , ( "parent", Encode.string node.parent )
        , ( "points", Point.encodeList node.points )
        , ( "edgePoints", Point.encodeList node.edgePoints )
        ]


encodeNotification : Notification -> Encode.Value
encodeNotification not =
    Encode.object
        [ ( "id", Encode.string not.id )
        , ( "sourceNode", Encode.string not.sourceNode )
        , ( "subject", Encode.string not.subject )
        , ( "message", Encode.string not.message )
        ]


encodeNodeMove : NodeMove -> Encode.Value
encodeNodeMove nodeMove =
    Encode.object
        [ ( "id", Encode.string nodeMove.id )
        , ( "oldParent", Encode.string nodeMove.oldParent )
        , ( "newParent", Encode.string nodeMove.newParent )
        ]


encodeNodeCopy : NodeCopy -> Encode.Value
encodeNodeCopy nodeCopy =
    Encode.object
        [ ( "id", Encode.string nodeCopy.id )
        , ( "oldParent", Encode.string nodeCopy.oldParent )
        , ( "newParent", Encode.string nodeCopy.newParent )
        , ( "duplicate", Encode.bool nodeCopy.duplicate )
        ]


encodeNodeDelete : NodeDelete -> Encode.Value
encodeNodeDelete nodeDelete =
    Encode.object
        [ ( "parent", Encode.string nodeDelete.parent )
        ]


description : Node -> String
description d =
    case Point.get d.points Point.typeDescription "" of
        Just point ->
            point.text

        Nothing ->
            ""


getBestDesc : Node -> String
getBestDesc n =
    Point.getBestDesc n.points


delete :
    { token : String
    , id : String
    , parent : String
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
delete options =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = encodeNodeDelete { parent = options.parent } |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


insert :
    { token : String
    , node : Node
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
insert options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.node.id ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.node |> encode |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


{-| GeneratedToken is the reply to a key request on an enrollment token
node. The token is shown once; only its hash is stored.
-}
type alias GeneratedToken =
    { token : String
    }


decodeGeneratedToken : Decode.Decoder GeneratedToken
decodeGeneratedToken =
    Decode.succeed GeneratedToken
        |> required "token" Decode.string


{-| generateKey makes a token for an enrollToken node. The hash is stored on
the node; the token comes back once.
-}
generateKey :
    { token : String
    , id : String
    , onResponse : Data GeneratedToken -> msg
    }
    -> Cmd msg
generateKey options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "key" ] []
        , expect = Api.Data.expectJson options.onResponse decodeGeneratedToken
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


notify :
    { token : String
    , not : Notification
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
notify options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.not.sourceNode, "not" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.not |> encodeNotification |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


move :
    { token : String
    , id : String
    , oldParent : String
    , newParent : String
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
move options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "parents" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body =
            { id = options.id
            , oldParent = options.oldParent
            , newParent = options.newParent
            }
                |> encodeNodeMove
                |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


copy :
    { token : String
    , id : String
    , oldParent : String
    , newParent : String
    , duplicate : Bool
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
copy options =
    Http.request
        { method = "PUT"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "parents" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body =
            { id = options.id
            , oldParent = options.oldParent
            , newParent = options.newParent
            , duplicate = options.duplicate
            }
                |> encodeNodeCopy
                |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }
