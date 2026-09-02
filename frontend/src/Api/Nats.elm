port module Api.Nats exposing
    ( Command(..)
    , Event(..)
    , receive
    , send
    )

{-| The port protocol between Elm and the NATS connection in main.js.

JavaScript owns the connection; Elm owns the tree. Elm sends commands to
connect, fetch a subtree, say which subjects to watch, and write points.
JavaScript sends back connection state, fetched nodes, and live points,
batched per animation frame.

-}

import Api.Node as Node exposing (Node)
import Api.Point as Point exposing (Point)
import Json.Decode as Decode
import Json.Encode as Encode


port natsOut : Encode.Value -> Cmd msg


port natsIn : (Decode.Value -> msg) -> Sub msg


type Command
    = Connect String
    | Disconnect
    | Fetch { anchor : String, parent : String, id : String, depth : Int }
    | Watch (List String)
    | SendPoints { anchor : String, id : String, points : List Point }


type Event
    = Connected { userId : String, anchors : List String }
    | Disconnected
    | AuthFailed
    | Nodes { anchor : String, parent : String, id : String, depth : Int, nodes : List Node }
    | Points (List { nodeId : String, points : List Point })
    | EdgePoints (List { nodeId : String, parentId : String, points : List Point })
    | Error String


send : Command -> Cmd msg
send cmd =
    natsOut (encodeCommand cmd)


receive : (Result Decode.Error Event -> msg) -> Sub msg
receive toMsg =
    natsIn (Decode.decodeValue decodeEvent >> toMsg)


encodeCommand : Command -> Encode.Value
encodeCommand cmd =
    case cmd of
        Connect token ->
            Encode.object
                [ ( "cmd", Encode.string "connect" )
                , ( "token", Encode.string token )
                ]

        Disconnect ->
            Encode.object [ ( "cmd", Encode.string "disconnect" ) ]

        Fetch f ->
            Encode.object
                [ ( "cmd", Encode.string "fetch" )
                , ( "anchor", Encode.string f.anchor )
                , ( "parent", Encode.string f.parent )
                , ( "id", Encode.string f.id )
                , ( "depth", Encode.int f.depth )
                ]

        Watch subjects ->
            Encode.object
                [ ( "cmd", Encode.string "watch" )
                , ( "subjects", Encode.list Encode.string subjects )
                ]

        SendPoints s ->
            Encode.object
                [ ( "cmd", Encode.string "sendPoints" )
                , ( "anchor", Encode.string s.anchor )
                , ( "id", Encode.string s.id )
                , ( "points", Point.encodeList s.points )
                ]


decodeEvent : Decode.Decoder Event
decodeEvent =
    Decode.field "event" Decode.string
        |> Decode.andThen
            (\event ->
                case event of
                    "connected" ->
                        Decode.map2 (\userId anchors -> Connected { userId = userId, anchors = anchors })
                            (Decode.field "userId" Decode.string)
                            (Decode.field "anchors" (Decode.list Decode.string))

                    "disconnected" ->
                        Decode.succeed Disconnected

                    "authFailed" ->
                        Decode.succeed AuthFailed

                    "nodes" ->
                        Decode.map5
                            (\anchor parent id depth nodes ->
                                Nodes { anchor = anchor, parent = parent, id = id, depth = depth, nodes = nodes }
                            )
                            (Decode.field "anchor" Decode.string)
                            (Decode.field "parent" Decode.string)
                            (Decode.field "id" Decode.string)
                            (Decode.field "depth" Decode.int)
                            (Decode.field "nodes" (Decode.list Node.decode))

                    "points" ->
                        Decode.map Points
                            (Decode.field "items"
                                (Decode.list
                                    (Decode.map2 (\nodeId points -> { nodeId = nodeId, points = points })
                                        (Decode.field "nodeId" Decode.string)
                                        (Decode.field "points" (Decode.list Point.decode))
                                    )
                                )
                            )

                    "edgePoints" ->
                        Decode.map EdgePoints
                            (Decode.field "items"
                                (Decode.list
                                    (Decode.map3
                                        (\nodeId parentId points ->
                                            { nodeId = nodeId, parentId = parentId, points = points }
                                        )
                                        (Decode.field "nodeId" Decode.string)
                                        (Decode.field "parentId" Decode.string)
                                        (Decode.field "points" (Decode.list Point.decode))
                                    )
                                )
                            )

                    "error" ->
                        Decode.map Error (Decode.field "message" Decode.string)

                    _ ->
                        Decode.fail ("unknown event " ++ event)
            )
