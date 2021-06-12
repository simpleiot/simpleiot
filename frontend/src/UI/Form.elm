module UI.Form exposing
    ( button
    , buttonRow
    , label
    , nodeCheckboxInput
    , nodeCounterWithReset
    , nodeNumberInput
    , nodeOnOffInput
    , nodeOptionInput
    , nodeTextInput
    , onEnter
    , onEnterEsc
    , onEsc
    , viewTextProperty
    )

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Color
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Html.Events
import Json.Decode as Decode
import Round
import Svg as S
import Svg.Attributes as Sa
import Time
import UI.Sanitize as Sanitize
import UI.Style as Style


onEnter : msg -> Element.Attribute msg
onEnter msg =
    Element.htmlAttribute
        (Html.Events.on "keyup"
            (Decode.field "key" Decode.string
                |> Decode.andThen
                    (\key ->
                        if key == "Enter" then
                            Decode.succeed msg

                        else
                            Decode.fail "Not the enter key"
                    )
            )
        )


onEnterEsc : msg -> msg -> Element.Attribute msg
onEnterEsc enterMsg escMsg =
    Element.htmlAttribute
        (Html.Events.on "keyup"
            (Decode.field "key" Decode.string
                |> Decode.andThen
                    (\key ->
                        if key == "Enter" then
                            Decode.succeed enterMsg

                        else if key == "Escape" then
                            Decode.succeed escMsg

                        else
                            Decode.fail "Not the enter key"
                    )
            )
        )


onEsc : msg -> Element.Attribute msg
onEsc msg =
    Element.htmlAttribute
        (Html.Events.on "keyup"
            (Decode.field "key" Decode.string
                |> Decode.andThen
                    (\key ->
                        if key == "Escape" then
                            Decode.succeed msg

                        else
                            Decode.fail "Not the esc key"
                    )
            )
        )


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
    { onEditNodePoint : Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
    }
    -> String
    -> Int
    -> String
    -> String
    -> Element msg
nodeTextInput o id index typ lbl =
    Input.text
        []
        { onChange =
            \d ->
                o.onEditNodePoint (Point id index typ o.now 0 d 0 0)
        , text = Point.getText o.node.points id index typ
        , placeholder = Nothing
        , label = Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeCheckboxInput :
    { onEditNodePoint : Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
    }
    -> String
    -> Int
    -> String
    -> String
    -> Element msg
nodeCheckboxInput o id index typ lbl =
    Input.checkbox
        []
        { onChange =
            \d ->
                let
                    v =
                        if d then
                            1.0

                        else
                            0.0
                in
                o.onEditNodePoint
                    (Point id index typ o.now v "" 0 0)
        , checked =
            Point.getValue o.node.points id index typ == 1
        , icon = Input.defaultCheckbox
        , label = Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeNumberInput :
    { onEditNodePoint : Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
    }
    -> String
    -> Int
    -> String
    -> String
    -> Element msg
nodeNumberInput o id index typ lbl =
    let
        pMaybe =
            Point.get o.node.points id index typ

        currentValue =
            case pMaybe of
                Just p ->
                    if p.text /= "" then
                        if p.text == Point.blankMajicValue || p.text == "blank" then
                            ""

                        else
                            Sanitize.float p.text

                    else
                        String.fromFloat (Round.roundNum 6 p.value)

                Nothing ->
                    ""

        currentValueF =
            case pMaybe of
                Just p ->
                    p.value

                Nothing ->
                    0
    in
    Input.text
        []
        { onChange =
            \d ->
                let
                    dCheck =
                        if d == "" then
                            Point.blankMajicValue

                        else
                            case String.toFloat d of
                                Just _ ->
                                    d

                                Nothing ->
                                    currentValue

                    v =
                        if dCheck == Point.blankMajicValue then
                            0

                        else
                            Maybe.withDefault currentValueF <| String.toFloat dCheck
                in
                o.onEditNodePoint
                    (Point id index typ o.now v dCheck 0 0)
        , text = currentValue
        , placeholder = Nothing
        , label = Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeOptionInput :
    { onEditNodePoint : Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
    }
    -> String
    -> Int
    -> String
    -> String
    -> List ( String, String )
    -> Element msg
nodeOptionInput o id index typ lbl options =
    Input.radio
        [ spacing 6 ]
        { onChange =
            \sel ->
                o.onEditNodePoint
                    (Point id index typ o.now 0 sel 0 0)
        , label =
            Input.labelLeft [ padding 12, width (px o.labelWidth) ] <|
                el [ alignRight ] <|
                    text <|
                        lbl
                            ++ ":"
        , selected = Just <| Point.getText o.node.points id index typ
        , options =
            List.map
                (\opt ->
                    Input.option (Tuple.first opt) (text (Tuple.second opt))
                )
                options
        }


nodeCounterWithReset :
    { onEditNodePoint : Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
    }
    -> String
    -> Int
    -> String
    -> String
    -> String
    -> Element msg
nodeCounterWithReset o id index typ pointResetName lbl =
    let
        currentValue =
            Point.getValue o.node.points id index typ

        currentResetValue =
            Point.getValue o.node.points id index pointResetName /= 0
    in
    row [ spacing 20 ]
        [ el [ width (px o.labelWidth) ] <|
            el [ alignRight ] <|
                text <|
                    lbl
                        ++ ": "
                        ++ String.fromFloat currentValue
        , Input.checkbox []
            { onChange =
                \v ->
                    let
                        vFloat =
                            if v then
                                1.0

                            else
                                0
                    in
                    o.onEditNodePoint (Point id index pointResetName o.now vFloat "" 0 0)
            , icon = Input.defaultCheckbox
            , checked = currentResetValue
            , label =
                Input.labelLeft [] (text "reset")
            }
        ]


nodeOnOffInput :
    { onEditNodePoint : Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
    }
    -> String
    -> Int
    -> String
    -> String
    -> String
    -> Element msg
nodeOnOffInput o id index typ pointSetName lbl =
    let
        currentValue =
            Point.getValue o.node.points id index typ

        currentSetValue =
            Point.getValue o.node.points id index pointSetName

        fill =
            if currentSetValue == 0 then
                Color.rgb 0.5 0.5 0.5

            else
                Color.rgb255 50 100 150

        fillFade =
            if currentSetValue == 0 then
                Color.rgb 0.9 0.9 0.9

            else
                Color.rgb255 150 200 255

        fillFadeS =
            Color.toCssString fillFade

        fillS =
            Color.toCssString fill

        offset =
            if currentSetValue == 0 then
                3

            else
                3 + 24

        newValue =
            if currentSetValue == 0 then
                1

            else
                0
    in
    row [ spacing 10 ]
        [ el [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        , Input.button
            []
            { onPress = Just <| o.onEditNodePoint (Point id index pointSetName o.now newValue "" 0 0)
            , label =
                el [ width (px 100) ] <|
                    html <|
                        S.svg [ Sa.viewBox "0 0 48 24" ]
                            [ S.rect
                                [ Sa.x "0"
                                , Sa.y "0"
                                , Sa.width "48"
                                , Sa.height "24"
                                , Sa.ry "3"
                                , Sa.rx "3"
                                , Sa.fill fillS
                                ]
                              <|
                                if currentValue /= currentSetValue then
                                    [ S.animate
                                        [ Sa.attributeName "fill"
                                        , Sa.dur "2s"
                                        , Sa.repeatCount "indefinite"
                                        , Sa.values <|
                                            fillFadeS
                                                ++ ";"
                                                ++ fillS
                                                ++ ";"
                                                ++ fillFadeS
                                        ]
                                        []
                                    ]

                                else
                                    []
                            , S.rect
                                [ Sa.x (String.fromFloat offset)
                                , Sa.y "3"
                                , Sa.width "18"
                                , Sa.height "18"
                                , Sa.ry "3"
                                , Sa.rx "3"
                                , Sa.fill (Color.toCssString Color.white)
                                ]
                                []
                            ]
            }
        ]
