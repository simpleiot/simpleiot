module Components.NodeGroup exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Border as Border
import Element.Input as Input
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view :
    { isRoot : Bool
    , now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , node : Node
    , onApiDelete : String -> msg
    , onEditNodePoint : String -> Point -> msg
    , onDiscardEdits : msg
    , onApiPostPoints : String -> msg
    }
    -> Element msg
view o =
    let
        textInput2 =
            textInput { onEditNodePoint = o.onEditNodePoint, node = o.node, now = o.now }
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
        [ wrappedRow [ spacing 10 ]
            [ Icon.users
            , text <|
                Point.getPointText o.node.points Point.typeDescription
            , viewIf o.isRoot <|
                Icon.x (o.onApiDelete o.node.id)
            ]
        , textInput2 Point.typeDescription "Description"
        , viewIf o.modified <|
            Form.buttonRow
                [ Form.button
                    { label = "save"
                    , color = colors.blue
                    , onPress = o.onApiPostPoints o.node.id
                    }
                , Form.button
                    { label = "discard"
                    , color = colors.gray
                    , onPress = o.onDiscardEdits
                    }
                ]
        ]


textInput :
    { onEditNodePoint : String -> Point -> msg
    , node : Node
    , now : Time.Posix
    }
    -> String
    -> String
    -> Element msg
textInput o pointName label =
    Input.text
        []
        { onChange =
            \d ->
                o.onEditNodePoint o.node.id
                    (Point "" pointName 0 o.now 0 d 0 0)
        , text = Point.getPointText o.node.points pointName
        , placeholder = Nothing
        , label = Input.labelLeft [ width (px 100) ] <| el [ alignRight ] <| text <| label ++ ":"
        }
