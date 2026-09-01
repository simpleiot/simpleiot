module Components.NodeOptions exposing (CopyMove(..), GeneratedKey, NodeOptions, findNode, oToInputO)

import Api.Node exposing (Node, NodeView)
import Api.Point exposing (Point)
import Time
import Tree exposing (Tree)
import Tree.Zipper as Zipper
import UI.NodeInputs exposing (NodeInputOptions)


type CopyMove
    = CopyMoveNone
      -- ID, source, description
    | Copy String String String


{-| GeneratedKey is a key the upstream just made for a credential node. The
seed is shown once, on that node, until the page is left.
-}
type alias GeneratedKey =
    { id : String
    , pubKey : String
    , seed : String
    }


type alias NodeOptions msg =
    { now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , children : List NodeView
    , nodes : List (Tree NodeView)
    , onEditNodePoint : List Point -> msg
    , onEditScratch : String -> msg
    , onUploadFile : Bool -> msg
    , onGenerateKey : msg
    , onInstallDeviceKey : String -> msg
    , generatedKey : Maybe GeneratedKey
    , copy : CopyMove
    , scratch : String
    }


oToInputO : NodeOptions msg -> Int -> NodeInputOptions msg
oToInputO o labelWidth =
    { onEditNodePoint = o.onEditNodePoint
    , onEditScratch = o.onEditScratch
    , node = o.node
    , now = o.now
    , zone = o.zone
    , labelWidth = labelWidth
    , scratch = o.scratch
    }


findNodeTree : Tree NodeView -> String -> Maybe NodeView
findNodeTree tree id =
    Zipper.findFromRoot (\n -> n.node.id == id) (Zipper.fromTree tree)
        |> Maybe.map Zipper.label


findNode : List (Tree NodeView) -> String -> Maybe Node
findNode nodes id =
    List.foldl
        (\t ret ->
            case findNodeTree t id of
                Just found ->
                    Just found.node

                Nothing ->
                    ret
        )
        Nothing
        nodes
