module Components.NodeUser exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Border as Border
import Element.Input as Input
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style exposing (colors, size)


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
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        [ wrappedRow [ spacing 10 ]
            [ viewNodeId o.node.id
            , if o.isRoot then
                Icon.x (o.onApiDelete o.node.id)

              else
                Element.none
            ]
        , Input.text
            []
            { onChange =
                \d ->
                    o.onEditNodePoint o.node.id
                        (Point "" Point.typeFirstName 0 o.now 0 d 0 0)
            , text = Point.getPointText o.node.points Point.typeFirstName
            , placeholder = Just <| Input.placeholder [] <| text "first name"
            , label = Input.labelLeft [] <| text "First Name:"
            }
        , Input.text
            []
            { onChange =
                \d ->
                    o.onEditNodePoint o.node.id
                        (Point "" Point.typeLastName 0 o.now 0 d 0 0)
            , text = Point.getPointText o.node.points Point.typeLastName
            , placeholder = Just <| Input.placeholder [] <| text "last name"
            , label = Input.labelLeft [] <| text "Last Name:"
            }
        , Input.text
            []
            { onChange =
                \d ->
                    o.onEditNodePoint o.node.id
                        (Point "" Point.typeEmail 0 o.now 0 d 0 0)
            , text = Point.getPointText o.node.points Point.typeEmail
            , placeholder = Just <| Input.placeholder [] <| text "email"
            , label = Input.labelLeft [] <| text "Email:"
            }
        , Input.text
            []
            { onChange =
                \d ->
                    o.onEditNodePoint o.node.id
                        (Point "" Point.typePhone 0 o.now 0 d 0 0)
            , text = Point.getPointText o.node.points Point.typePhone
            , placeholder = Just <| Input.placeholder [] <| text "phone number"
            , label = Input.labelLeft [] <| text "Phone:"
            }
        , Input.text
            []
            { onChange =
                \d ->
                    o.onEditNodePoint o.node.id
                        (Point "" Point.typePass 0 o.now 0 d 0 0)
            , text = Point.getPointText o.node.points Point.typePass
            , placeholder = Just <| Input.placeholder [] <| text "password"
            , label = Input.labelLeft [] <| text "Password:"
            }
        ]
            ++ (if o.modified then
                    [ Form.buttonRow
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

                else
                    []
               )


viewNodeId : String -> Element msg
viewNodeId id =
    el
        [ padding 16
        , size.heading
        ]
    <|
        text id
