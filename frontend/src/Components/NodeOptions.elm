module Components.NodeOptions exposing (NodeOptions)

import Api.Node exposing (Node)
import Api.Point exposing (Point)
import Time


type alias NodeOptions msg =
    { now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onEditNodePoint : Point -> msg
    }
