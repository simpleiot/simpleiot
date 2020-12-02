module UI.Form exposing
    ( button
    , buttonRow
    , label
    , nodeNumberInput
    , nodeOptionInput
    , nodeTextInput
    , viewTextProperty
    )

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Time
import UI.Style as Style


type alias TextProperty msg =
    { name : String
    , value : String
    , action : String -> msg
    }


viewTextProperty : TextProperty msg -> Element msg
viewTextProperty { name, value, action } =
    Input.text
        [ padding 16
        , width (fill |> minimum 150)
        , Border.width 0
        , Border.rounded 0
        , Background.color Style.colors.pale
        , spacing 0
        ]
        { onChange = action
        , text = value
        , placeholder = Nothing
        , label = label Input.labelAbove name
        }


label : (List (Attribute msg) -> Element msg -> Input.Label msg) -> (String -> Input.Label msg)
label kind =
    kind
        [ padding 16
        , Font.italic
        , Font.color Style.colors.gray
        ]
        << text


buttonRow : List (Element msg) -> Element msg
buttonRow =
    row
        [ Font.size 16
        , Font.bold
        , width fill
        , padding 16
        , spacing 16
        ]


button :
    { color : Color
    , onPress : msg
    , label : String
    }
    -> Element msg
button options =
    Input.button
        (Style.button options.color)
        { onPress = Just options.onPress
        , label = el [ centerX ] <| text options.label
        }


nodeTextInput :
    { onEditNodePoint : String -> Point -> msg
    , node : Node
    , now : Time.Posix
    }
    -> String
    -> String
    -> Element msg
nodeTextInput o pointName lbl =
    Input.text
        []
        { onChange =
            \d ->
                o.onEditNodePoint o.node.id
                    (Point "" pointName 0 o.now 0 d 0 0)
        , text = Point.getPointText o.node.points pointName
        , placeholder = Nothing
        , label = Input.labelLeft [ width (px 100) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeNumberInput :
    { onEditNodePoint : String -> Point -> msg
    , node : Node
    , now : Time.Posix
    }
    -> String
    -> String
    -> Element msg
nodeNumberInput o pointName lbl =
    let
        currentValue =
            Point.getPointValue o.node.points pointName
    in
    Input.text
        []
        { onChange =
            \d ->
                let
                    v =
                        if d == "" then
                            0

                        else
                            Maybe.withDefault currentValue <| String.toFloat d
                in
                o.onEditNodePoint o.node.id
                    (Point "" pointName 0 o.now v "" 0 0)
        , text =
            if currentValue == 0 then
                ""

            else
                String.fromFloat currentValue
        , placeholder = Nothing
        , label = Input.labelLeft [ width (px 100) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeOptionInput :
    { onEditNodePoint : String -> Point -> msg
    , node : Node
    , now : Time.Posix
    }
    -> String
    -> String
    -> List ( String, String )
    -> Element msg
nodeOptionInput o pointName lbl options =
    Input.radio
        [ spacing 6 ]
        { onChange =
            \sel ->
                o.onEditNodePoint o.node.id
                    (Point "" pointName 0 o.now 0 sel 0 0)
        , label =
            Input.labelLeft [ padding 12, width (px 100) ] <|
                el [ alignRight ] <|
                    text <|
                        lbl
                            ++ ":"
        , selected = Just <| Point.getPointText o.node.points pointName
        , options =
            List.map
                (\opt ->
                    Input.option (Tuple.first opt) (text (Tuple.second opt))
                )
                options
        }
