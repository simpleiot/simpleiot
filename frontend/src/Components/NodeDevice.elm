module Components.NodeDevice exposing (view)

import Api.Node as Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Input as Input
import Time
import UI.Icon as Icon
import UI.Style as Style exposing (colors)
import UI.ViewIf exposing (viewIf)
import Utils.Duration as Duration
import Utils.Iso8601 as Iso8601


view :
    { isRoot : Bool
    , now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onEditNodePoint : Point -> msg
    }
    -> Element msg
view o =
    let
        sysState =
            Point.getText o.node.points "" 0 Point.typeSysState

        sysStateIcon =
            case sysState of
                -- not sure why I can't use defines in Node.elm here
                "powerOff" ->
                    Icon.power

                "offline" ->
                    Icon.cloudOff

                "online" ->
                    Icon.cloud

                _ ->
                    Element.none

        background =
            case sysState of
                "online" ->
                    Style.colors.white

                _ ->
                    Style.colors.gray

        hwVersion =
            case Point.get o.node.points "" 0 Point.typeHwVersion of
                Just point ->
                    "HW: " ++ point.text

                Nothing ->
                    ""

        osVersion =
            case Point.get o.node.points "" 0 Point.typeOSVersion of
                Just point ->
                    "OS: " ++ point.text

                Nothing ->
                    ""

        appVersion =
            case Point.get o.node.points "" 0 Point.typeAppVersion of
                Just point ->
                    "App: " ++ point.text

                Nothing ->
                    ""

        latestPointTime =
            case Point.getLatest o.node.points of
                Just point ->
                    point.time

                Nothing ->
                    Time.millisToPosix 0
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , Background.color background
        , spacing 6
        ]
    <|
        wrappedRow
            [ spacing 10 ]
            [ Icon.device
            , sysStateIcon
            , Input.text
                [ Background.color background ]
                { onChange =
                    \d ->
                        o.onEditNodePoint
                            (Point "" 0 Point.typeDescription o.now 0 d 0 0)
                , text = Node.description o.node
                , placeholder = Just <| Input.placeholder [] <| text "node description"
                , label = Input.labelHidden "node description"
                }
            ]
            :: (if o.expDetail then
                    [ viewPoints <| Point.filterSpecialPoints o.node.points
                    , text ("Last update: " ++ Iso8601.toDateTimeString o.zone latestPointTime)
                    , text
                        ("Time since last update: "
                            ++ Duration.toString
                                (Time.posixToMillis o.now
                                    - Time.posixToMillis latestPointTime
                                )
                        )
                    , viewIf (hwVersion /= "" && osVersion /= "" && appVersion /= "") <|
                        text
                            ("Version: "
                                ++ hwVersion
                                ++ " "
                                ++ osVersion
                                ++ " "
                                ++ appVersion
                            )
                    ]

                else
                    []
               )


viewPoints : List Point.Point -> Element msg
viewPoints ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (Point.renderPoint >> text) ios
