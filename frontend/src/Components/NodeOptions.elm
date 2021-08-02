module Components.NodeOptions exposing (NodeOptions, oToInputO)

import Api.Node exposing (Node)
import Api.Point exposing (Point)
import Time
import UI.Form exposing (NodeInputOptions)


type alias NodeOptions msg =
    { now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onEditNodePoint : List Point -> msg
    }


oToInputO : NodeOptions msg -> Int -> NodeInputOptions msg
oToInputO o labelWidth =
    { onEditNodePoint = o.onEditNodePoint
    , node = o.node
    , now = o.now
    , zone = o.zone
    , labelWidth = labelWidth
    }
