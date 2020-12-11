module UI.Form exposing
    ( button
    , buttonRow
    , label
    , nodeNumberInput
    , nodeOnOffInput
    , nodeOptionInput
    , nodeTextInput
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
import Time
import TypedSvg as S
import TypedSvg.Attributes as Sa
import TypedSvg.Types as St
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
    , labelWidth : Int
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
        , text = Point.getText o.node.points pointName
        , placeholder = Nothing
        , label = Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeNumberInput :
    { onEditNodePoint : String -> Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
    }
    -> String
    -> String
    -> Element msg
nodeNumberInput o pointName lbl =
    let
        pMaybe =
            Point.get o.node.points "" pointName 0

        currentValue =
            case pMaybe of
                Just p ->
                    if p.text /= "" then
                        if p.text == Point.blankMajicValue || p.text == "blank" then
                            ""

                        else
                            p.text

                    else
                        String.fromFloat p.value

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
                o.onEditNodePoint o.node.id
                    (Point "" pointName 0 o.now v dCheck 0 0)
        , text = currentValue
        , placeholder = Nothing
        , label = Input.labelLeft [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        }


nodeOptionInput :
    { onEditNodePoint : String -> Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
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
            Input.labelLeft [ padding 12, width (px o.labelWidth) ] <|
                el [ alignRight ] <|
                    text <|
                        lbl
                            ++ ":"
        , selected = Just <| Point.getText o.node.points pointName
        , options =
            List.map
                (\opt ->
                    Input.option (Tuple.first opt) (text (Tuple.second opt))
                )
                options
        }


nodeOnOffInput :
    { onEditNodePoint : String -> Point -> msg
    , node : Node
    , now : Time.Posix
    , labelWidth : Int
    }
    -> String
    -> String
    -> Element msg
nodeOnOffInput o pointName lbl =
    let
        currentValue =
            Point.getValue o.node.points pointName

        buttonColor =
            if currentValue == 0 then
                Color.rgb 0.5 0.5 0.5

            else
                Color.rgb255 50 100 150

        offset =
            if currentValue == 0 then
                3

            else
                3 + 24

        newValue =
            if currentValue == 0 then
                1

            else
                0
    in
    row [ spacing 10 ]
        [ el [ width (px o.labelWidth) ] <| el [ alignRight ] <| text <| lbl ++ ":"
        , Input.button
            []
            { onPress = Just <| o.onEditNodePoint o.node.id (Point "" pointName 0 o.now newValue "" 0 0)
            , label =
                el [ width (px 100) ] <|
                    html <|
                        S.svg [ Sa.viewBox 0 0 48 24 ]
                            [ S.rect
                                [ Sa.x (St.px 0)
                                , Sa.y (St.px 0)
                                , Sa.width (St.px 48)
                                , Sa.height (St.px 24)
                                , Sa.ry (St.px 3)
                                , Sa.rx (St.px 3)
                                , Sa.fill <| St.Paint <| buttonColor
                                ]
                                []
                            , S.rect
                                [ Sa.x (St.px offset)
                                , Sa.y (St.px 3)
                                , Sa.width (St.px 18)
                                , Sa.height (St.px 18)
                                , Sa.ry (St.px 3)
                                , Sa.rx (St.px 3)
                                , Sa.fill <| St.Paint <| Color.white
                                ]
                                []
                            ]
            }
        ]
